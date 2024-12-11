variable "aws_region" {
  default = "eu-west-2"
}

variable "domain_name" {
  default = "apiprod.lewislewin.dev"
}

variable "cloudflare_zone_id" {
    default = "283afa66c3086afe7e6e4471c8e3e21a"
}

variable "cloudflare_api_token" {
    default = "p59IWVHw8iN_kuNU8ykpYm741bfeQ1fAsQH73jdX"
}

variable "db_username" {
  default = "postgres"
}
variable "db_name" {
  default = "receipts"
}

variable "jwt_secret_name" {
  default = "receipt_jwt_secret"
}

variable "google_api_key_name" {
  default = "google_api_key"
}

variable "openapi_api_key_name" {
  default = "openapi_api_key"
}

variable "openapi_project_key_name" {
  default = "openapi_project_key"
}

variable "openapi_organisation_key_name" {
  default = "openapi_organisation_key"
}

variable "db_password_name" {
  default = "receipt_db_password"
}

variable "ecr_image_url" {
  default = "169498589229.dkr.ecr.eu-west-2.amazonaws.com/receipt-splitter-backend:latest"
}

variable "github_repository" {
  default = "lewislewin/receipt-splitter-backend"
}
