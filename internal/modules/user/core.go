package user

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"

	"github.com/Laaaaksh/gohighlevel-round1/internal/logger"
	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/user/entities"
	"github.com/Laaaaksh/gohighlevel-round1/pkg/apperror"
	"github.com/Laaaaksh/gohighlevel-round1/pkg/idgen"
)

// bcryptCost is deliberately the library default, not tuned down for
// speed: registration is the one write path allowed to be slow (that is
// the whole point of bcrypt), and it must never be reused on a read path.
const bcryptCost = bcrypt.DefaultCost

// ICore is the user module's business logic, free of HTTP concerns. Exists
// is the narrow read other modules (post, follow) depend on to validate a
// userId - see their repository.go files for the structural interface each
// declares for it; Go's structural typing means they never import user.
type ICore interface {
	Register(ctx context.Context, req entities.RegisterUserRequest) (*entities.RegisterUserResponse, error)
	Exists(ctx context.Context, userID string) (bool, error)
}

// Core implements ICore against an IRepository. now is injected so tests
// can fix the clock instead of racing time.Now() against gomock's
// exact-argument matching.
type Core struct {
	repo   IRepository
	now    func() time.Time
	newID  func() (string, error)
	hasher func(password []byte, cost int) ([]byte, error)
}

var _ ICore = (*Core)(nil)

func NewCore(repo IRepository) *Core {
	return newCore(repo, time.Now, idgen.New, bcrypt.GenerateFromPassword)
}

// NewCoreWithClock lets a test fix the clock and id generator - see the
// mock-import-cycle note in AGENTS.md for why core_test.go lives in an
// external package and needs this seam instead of Core's unexported fields.
func NewCoreWithClock(repo IRepository, now func() time.Time, newID func() (string, error)) *Core {
	return newCore(repo, now, newID, bcrypt.GenerateFromPassword)
}

func newCore(
	repo IRepository,
	now func() time.Time,
	newID func() (string, error),
	hasher func(password []byte, cost int) ([]byte, error),
) *Core {
	return &Core{repo: repo, now: now, newID: newID, hasher: hasher}
}

func (c *Core) Register(ctx context.Context, req entities.RegisterUserRequest) (*entities.RegisterUserResponse, error) {
	dob, validationErr := validateRegisterFields(req, c.now())
	if validationErr != nil {
		return nil, validationErr
	}

	passwordHash, hashErr := c.hasher([]byte(req.Password), bcryptCost)
	if hashErr != nil {
		logger.Ctx(ctx).Error(logMsgHashPasswordFailed, logFieldError, hashErr)
		return nil, apperror.Wrap(apperror.CodeInternalError, apperror.MsgInternalError, hashErr)
	}

	userID, idErr := c.newID()
	if idErr != nil {
		logger.Ctx(ctx).Error(logMsgGenerateUserIDFailed, logFieldError, idErr)
		return nil, apperror.Wrap(apperror.CodeInternalError, apperror.MsgInternalError, idErr)
	}

	newUser := &User{
		ID:           userID,
		Name:         req.Name,
		Handle:       req.Handle,
		DOB:          dob,
		PasswordHash: string(passwordHash),
		CreatedAt:    c.now().UTC(),
	}

	if err := c.repo.Create(ctx, newUser); err != nil {
		if errors.Is(err, ErrHandleTaken) {
			return nil, handleTakenError()
		}
		logger.Ctx(ctx).Error(logMsgRegisterUserFailed, logFieldError, err)
		return nil, apperror.Wrap(apperror.CodeInternalError, apperror.MsgInternalError, err)
	}

	logger.Ctx(ctx).Info(logMsgUserRegistered, logFieldUserID, userID)
	return &entities.RegisterUserResponse{UserID: userID}, nil
}

// Exists is a read-path-only lookup: no password hash ever crosses this
// boundary, and it must stay index-served (see repository.go) to hold the
// service's 10ms user-lookup budget under load.
func (c *Core) Exists(ctx context.Context, userID string) (bool, error) {
	found, err := c.repo.ExistsByID(ctx, userID)
	if err != nil {
		logger.Ctx(ctx).Error(logMsgLookupUserFailed, logFieldError, err, logFieldUserID, userID)
		return false, apperror.Wrap(apperror.CodeInternalError, apperror.MsgInternalError, err)
	}
	return found, nil
}

// validateRegisterFields is the explicit write-path check behind Gin's
// binding tags. It also parses and age-checks dob, returning the parsed
// value so the caller does not parse the string twice.
func validateRegisterFields(req entities.RegisterUserRequest, now time.Time) (time.Time, *apperror.Error) {
	if strings.TrimSpace(req.Name) == "" {
		return time.Time{}, validationError(apperror.FieldName, apperror.MsgNameRequired)
	}
	if utf8.RuneCountInString(req.Name) > entities.MaxNameLength {
		return time.Time{}, validationError(apperror.FieldName, apperror.MsgNameTooLong)
	}
	if strings.TrimSpace(req.Handle) == "" {
		return time.Time{}, validationError(apperror.FieldHandle, apperror.MsgHandleRequired)
	}
	if strings.TrimSpace(req.Password) == "" {
		return time.Time{}, validationError(apperror.FieldPassword, apperror.MsgPasswordRequired)
	}

	if strings.TrimSpace(req.DOB) == "" {
		return time.Time{}, validationError(apperror.FieldDOB, apperror.MsgDOBRequired)
	}
	dob, err := time.Parse(entities.DOBLayout, req.DOB)
	if err != nil {
		return time.Time{}, validationError(apperror.FieldDOB, apperror.MsgDOBMalformed)
	}
	if !isAtLeast(dob, now.UTC(), entities.MinAgeYears) {
		return time.Time{}, validationError(apperror.FieldDOB, apperror.MsgDOBTooYoung)
	}
	return dob, nil
}

// isAtLeast reports whether dob is at least minYears before asOf, computed
// entirely in UTC. AddDate handles the 29 Feb case the way the calendar
// does: adding 1 non-leap year to a leap-day date rolls to 1 Mar, so a
// leap-day birthday's "anniversary" in a non-leap year is treated as
// already past by 1 Mar - it never panics and never off-by-ones the check.
func isAtLeast(dob, asOf time.Time, minYears int) bool {
	eighteenthBirthday := dob.AddDate(minYears, 0, 0)
	return !eighteenthBirthday.After(asOf)
}

func validationError(field, detail string) *apperror.Error {
	return apperror.New(apperror.CodeValidationError, apperror.MsgValidationFailed).WithField(field, detail)
}

func handleTakenError() *apperror.Error {
	return apperror.New(apperror.CodeConflict, apperror.MsgHandleTaken).WithField(apperror.FieldHandle, apperror.MsgHandleTaken)
}
