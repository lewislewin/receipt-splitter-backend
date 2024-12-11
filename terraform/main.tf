terraform {
  required_version = ">=0.14"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
    cloudflare = {
      source = "cloudflare/cloudflare"
      version = "~> 4.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

provider "cloudflare" {
  api_token = var.cloudflare_api_token
}

data "aws_secretsmanager_secret_version" "jwt_secret" {
  secret_id = var.jwt_secret_name
}

data "aws_secretsmanager_secret_version" "google_api_key" {
  secret_id = var.google_api_key_name
}

data "aws_secretsmanager_secret_version" "openapi_api_key" {
  secret_id = var.openapi_api_key_name
}

data "aws_secretsmanager_secret_version" "openapi_project_key" {
  secret_id = var.openapi_project_key_name
}

data "aws_secretsmanager_secret_version" "openapi_organisation_key" {
  secret_id = var.openapi_organisation_key_name
}

data "aws_secretsmanager_secret_version" "db_password" {
  secret_id = var.db_password_name
}

resource "aws_iam_openid_connect_provider" "github" {
  url             = "https://token.actions.githubusercontent.com"
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = ["9e99a48a9960f6e4f0a7d3d4c4987b7e7e34b8f2"]
}

resource "aws_iam_role" "github_deploy_role" {
  name = "github-deploy-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Effect = "Allow",
        Principal = {
          Federated = aws_iam_openid_connect_provider.github.arn
        },
        Action = "sts:AssumeRoleWithWebIdentity",
        Condition = {
          StringLike = {
            "token.actions.githubusercontent.com:sub" : "repo:${var.github_repository}:ref:refs/heads/main"
          }
        }
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "github_deploy_admin_attach" {
  role       = aws_iam_role.github_deploy_role.name
  policy_arn = "arn:aws:iam::aws:policy/AdministratorAccess"
}

resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_internet_gateway" "igw" {
  vpc_id = aws_vpc.main.id
}

# Public subnets (2 AZs)
resource "aws_subnet" "public_subnet_1" {
  vpc_id                  = aws_vpc.main.id
  cidr_block              = "10.0.1.0/24"
  map_public_ip_on_launch = true
  availability_zone       = "${var.aws_region}a"
}

resource "aws_subnet" "public_subnet_2" {
  vpc_id                  = aws_vpc.main.id
  cidr_block              = "10.0.2.0/24"
  map_public_ip_on_launch = true
  availability_zone       = "${var.aws_region}b"
}

# Private subnets (2 AZs)
resource "aws_subnet" "private_subnet_1" {
  vpc_id                  = aws_vpc.main.id
  cidr_block              = "10.0.3.0/24"
  map_public_ip_on_launch = false
  availability_zone       = "${var.aws_region}a"
}

resource "aws_subnet" "private_subnet_2" {
  vpc_id                  = aws_vpc.main.id
  cidr_block              = "10.0.4.0/24"
  map_public_ip_on_launch = false
  availability_zone       = "${var.aws_region}b"
}

resource "aws_route_table" "public_rt" {
  vpc_id = aws_vpc.main.id
}

resource "aws_route" "public_route" {
  route_table_id         = aws_route_table.public_rt.id
  destination_cidr_block = "0.0.0.0/0"
  gateway_id             = aws_internet_gateway.igw.id
}

resource "aws_route_table_association" "public_rta_1" {
  route_table_id = aws_route_table.public_rt.id
  subnet_id      = aws_subnet.public_subnet_1.id
}

resource "aws_route_table_association" "public_rta_2" {
  route_table_id = aws_route_table.public_rt.id
  subnet_id      = aws_subnet.public_subnet_2.id
}

resource "aws_eip" "nat_eip" {
  # Removed vpc = true (no longer needed)
}

resource "aws_nat_gateway" "natgw" {
  allocation_id = aws_eip.nat_eip.id
  subnet_id     = aws_subnet.public_subnet_1.id
}

resource "aws_route_table" "private_rt" {
  vpc_id = aws_vpc.main.id
}

resource "aws_route" "private_route" {
  route_table_id         = aws_route_table.private_rt.id
  destination_cidr_block = "0.0.0.0/0"
  nat_gateway_id         = aws_nat_gateway.natgw.id
}

resource "aws_route_table_association" "private_rta_1" {
  route_table_id = aws_route_table.private_rt.id
  subnet_id      = aws_subnet.private_subnet_1.id
}

resource "aws_route_table_association" "private_rta_2" {
  route_table_id = aws_route_table.private_rt.id
  subnet_id      = aws_subnet.private_subnet_2.id
}

resource "aws_security_group" "alb_sg" {
  name        = "alb_sg"
  description = "Allow HTTP/HTTPS inbound"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_security_group" "ecs_service_sg" {
  name        = "ecs_service_sg"
  description = "Allow inbound from ALB"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port       = 8080
    to_port         = 8080
    protocol        = "tcp"
    security_groups = [aws_security_group.alb_sg.id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_security_group" "db_sg" {
  name        = "db_sg"
  description = "Allow Postgres from ECS"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [aws_security_group.ecs_service_sg.id]
  }

  egress {
    from_port = 0
    to_port   = 0
    protocol  = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_db_subnet_group" "db_subnet_group" {
  name       = "receipt-splitter-db-subnet-group"
  subnet_ids = [
    aws_subnet.private_subnet_1.id,
    aws_subnet.private_subnet_2.id
  ]
}

resource "aws_db_instance" "db" {
  allocated_storage    = 20
  engine               = "postgres"
  engine_version       = "16.1"
  instance_class       = "db.t4g.micro"
  username             = var.db_username
  password             = data.aws_secretsmanager_secret_version.db_password.secret_string
  parameter_group_name = "default.postgres16"
  publicly_accessible  = false
  skip_final_snapshot  = true
  db_subnet_group_name = aws_db_subnet_group.db_subnet_group.name
  vpc_security_group_ids = [aws_security_group.db_sg.id]
}

resource "aws_iam_role" "ecs_task_execution_role" {
  name = "ecsTaskExecutionRole"

  assume_role_policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Action: "sts:AssumeRole",
        Principal = {
          Service = "ecs-tasks.amazonaws.com"
        },
        Effect = "Allow"
      }
    ]
  })
}

resource "aws_iam_policy_attachment" "ecs_task_execution_role_attachment" {
  name       = "ecs_task_execution_role_attachment"
  roles      = [aws_iam_role.ecs_task_execution_role.name]
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

resource "aws_iam_policy" "ecs_task_secrets_policy" {
  name        = "ecs_task_secrets_policy"
  description = "Allow ECS tasks to read secrets"
  policy      = jsonencode({
    Version = "2012-10-17",
    Statement: [
      {
        Action: [
          "secretsmanager:GetSecretValue"
        ],
        Effect: "Allow",
        Resource: "*"
      }
    ]
  })
}

resource "aws_iam_policy_attachment" "ecs_task_secrets_attach" {
  name       = "ecs_task_secrets_attach"
  roles      = [aws_iam_role.ecs_task_execution_role.name]
  policy_arn = aws_iam_policy.ecs_task_secrets_policy.arn
}

resource "aws_ecs_cluster" "this" {
  name = "receipt-splitter-cluster"
}

resource "aws_lb" "alb" {
  name               = "receipt-splitter-alb"
  load_balancer_type = "application"
  security_groups    = [aws_security_group.alb_sg.id]
  subnets            = [
    aws_subnet.public_subnet_1.id,
    aws_subnet.public_subnet_2.id
  ]
}

resource "aws_lb_target_group" "tg" {
  name     = "receipt-splitter-tg"
  port     = 8080
  protocol = "HTTP"
  vpc_id   = aws_vpc.main.id
  target_type = "ip"
}

resource "aws_acm_certificate" "cert" {
  domain_name       = var.domain_name
  validation_method = "DNS"
}

# Convert domain_validation_options to list
locals {
  dvo = tolist(aws_acm_certificate.cert.domain_validation_options)[0]
}

resource "cloudflare_record" "acm_validation" {
  zone_id = var.cloudflare_zone_id
  name    = local.dvo.resource_record_name
  type    = local.dvo.resource_record_type
  content = local.dvo.resource_record_value
  ttl     = 300
}

resource "aws_acm_certificate_validation" "cert_validation" {
  certificate_arn         = aws_acm_certificate.cert.arn
  validation_record_fqdns = [cloudflare_record.acm_validation.hostname]
}

resource "aws_lb_listener" "https_listener" {
  load_balancer_arn = aws_lb.alb.arn
  port              = 443
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-2016-08"
  certificate_arn   = aws_acm_certificate_validation.cert_validation.certificate_arn

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.tg.arn
  }
}

resource "cloudflare_record" "alb_dns" {
  zone_id = var.cloudflare_zone_id
  name    = var.domain_name
  type    = "CNAME"
  content = aws_lb.alb.dns_name
  ttl     = 300
  proxied = false
}

resource "aws_ecs_task_definition" "this" {
  family                   = "receipt-splitter-task"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = 512
  memory                   = 1024
  execution_role_arn       = aws_iam_role.ecs_task_execution_role.arn
  task_role_arn            = aws_iam_role.ecs_task_execution_role.arn

  container_definitions = jsonencode([
    {
      name      = "api",
      image     = var.ecr_image_url,
      essential = true,
      secrets = [
        { name = "DB_PASSWORD", valueFrom = data.aws_secretsmanager_secret_version.db_password.arn },
        { name = "JWT_SECRET", valueFrom = data.aws_secretsmanager_secret_version.jwt_secret.arn },
        { name = "GOOGLE_API_KEY", valueFrom = data.aws_secretsmanager_secret_version.google_api_key.arn },
        { name = "OPENAPI_API_KEY", valueFrom = data.aws_secretsmanager_secret_version.openapi_api_key.arn },
        { name = "OPENAPI_ORG_ID", valueFrom = data.aws_secretsmanager_secret_version.openapi_organisation_key.arn },
        { name = "OPENAPI_PROJECT_ID", valueFrom = data.aws_secretsmanager_secret_version.openapi_project_key.arn }
      ],
      environment = [
        { name = "DB_HOST", value = aws_db_instance.db.address },
        { name = "DB_USER", value = var.db_username },
        { name = "DB_NAME", value = var.db_name },
        { name = "DB_PORT", value = "5432" }
      ],
      portMappings = [
        {
          containerPort = 8080
          protocol      = "tcp"
        }
      ]
    }
  ])
}

resource "aws_ecs_service" "this" {
  name            = "receipt-splitter-service"
  cluster         = aws_ecs_cluster.this.id
  task_definition = aws_ecs_task_definition.this.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets         = [aws_subnet.private_subnet_1.id, aws_subnet.private_subnet_2.id]
    security_groups = [aws_security_group.ecs_service_sg.id]
    assign_public_ip = false
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.tg.arn
    container_name   = "api"
    container_port   = 8080
  }

  depends_on = [aws_acm_certificate_validation.cert_validation]
}
