// Mirrors internal/modules/item/entities/response.go and request.go on the
// Go backend. Keep the two in sync by hand when the resource shape changes.

export interface Item {
  id: string;
  name: string;
  description: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreateItemInput {
  name: string;
  description: string;
}
