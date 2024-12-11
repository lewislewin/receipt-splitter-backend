output "alb_dns_name" {
  value = aws_lb.alb.dns_name
}

output "service_url" {
  value = "https://${var.domain_name}"
}
