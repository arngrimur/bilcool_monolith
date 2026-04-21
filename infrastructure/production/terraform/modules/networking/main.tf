variable "prefix"     { type = string }
variable "aws_region" { type = string }

# ── VPC ───────────────────────────────────────────────────────────────────────

resource "aws_vpc" "main" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true
  tags                 = { Name = "${var.prefix}-vpc", Component = "networking" }
}

# ── Subnets ───────────────────────────────────────────────────────────────────

resource "aws_subnet" "public" {
  vpc_id                  = aws_vpc.main.id
  cidr_block              = "10.0.0.0/24"
  availability_zone       = "${var.aws_region}a"
  map_public_ip_on_launch = true
  tags                    = { Name = "${var.prefix}-public", Component = "networking" }
}

resource "aws_subnet" "private_a" {
  vpc_id            = aws_vpc.main.id
  cidr_block        = "10.0.1.0/24"
  availability_zone = "${var.aws_region}a"
  tags              = { Name = "${var.prefix}-private-a", Component = "networking" }
}

resource "aws_subnet" "private_b" {
  vpc_id            = aws_vpc.main.id
  cidr_block        = "10.0.2.0/24"
  availability_zone = "${var.aws_region}b"
  tags              = { Name = "${var.prefix}-private-b", Component = "networking" }
}

# ── Internet Gateway ──────────────────────────────────────────────────────────

resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id
  tags   = { Name = "${var.prefix}-igw", Component = "networking" }
}

# ── Elastic IP + NAT Gateway (in public subnet) ───────────────────────────────

resource "aws_eip" "nat" {
  domain = "vpc"
  tags   = { Name = "${var.prefix}-nat-eip", Component = "networking" }
}

resource "aws_nat_gateway" "main" {
  allocation_id = aws_eip.nat.id
  subnet_id     = aws_subnet.public.id
  tags          = { Name = "${var.prefix}-nat", Component = "networking" }
  depends_on    = [aws_internet_gateway.main]
}

# ── Route tables ──────────────────────────────────────────────────────────────

resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id
  tags   = { Name = "${var.prefix}-public-rt", Component = "networking" }

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.main.id
  }
}

resource "aws_route_table_association" "public" {
  subnet_id      = aws_subnet.public.id
  route_table_id = aws_route_table.public.id
}

resource "aws_route_table" "private" {
  vpc_id = aws_vpc.main.id
  tags   = { Name = "${var.prefix}-private-rt", Component = "networking" }

  route {
    cidr_block     = "0.0.0.0/0"
    nat_gateway_id = aws_nat_gateway.main.id
  }
}

resource "aws_route_table_association" "private_a" {
  subnet_id      = aws_subnet.private_a.id
  route_table_id = aws_route_table.private.id
}

resource "aws_route_table_association" "private_b" {
  subnet_id      = aws_subnet.private_b.id
  route_table_id = aws_route_table.private.id
}

# ── Lambda security group ─────────────────────────────────────────────────────

resource "aws_security_group" "lambda" {
  name        = "${var.prefix}-lambda"
  description = "Lambda functions egress only"
  vpc_id      = aws_vpc.main.id
  tags        = { Name = "${var.prefix}-lambda-sg", Component = "networking" }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

# ── outputs ───────────────────────────────────────────────────────────────────

output "private_subnet_ids"      { value = [aws_subnet.private_a.id, aws_subnet.private_b.id] }
output "lambda_sg_id"            { value = aws_security_group.lambda.id }
output "nat_gateway_public_ip"   { value = aws_eip.nat.public_ip }
