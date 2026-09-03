// Mirrors the schemas in docs/ru/openapi.yaml — field names match the JSON
// wire format (snake_case) directly rather than being remapped, since this
// is the only place in the frontend that needs to know that shape.

export interface Board {
  id: string;
  title: string;
  owner_id: string;
  created_at: string;
}

export interface BoardMember {
  user_id: string;
  email: string;
  name: string;
  joined_at: string;
}

export interface Card {
  id: string;
  column_id: string;
  title: string;
  description: string;
  order_num: number;
  author_id: string;
  created_at: string;
}

export interface Column {
  id: string;
  board_id: string;
  title: string;
  order_num: number;
  created_at: string;
}

export interface ColumnDetail extends Column {
  cards: Card[];
}

export interface BoardDetail {
  id: string;
  title: string;
  owner_id: string;
  columns: ColumnDetail[];
  members: BoardMember[];
  created_at: string;
}

export interface User {
  id: string;
  email: string;
  name: string;
  created_at: string;
}

export interface AuthResponse {
  token: string;
  user: User;
}
