terraform {
  required_providers {
    shell = {
      source = "scottwinkler/shell"
      version = "~> 1.7.0"
    }
  }
}

variable "database_url" {
  type = string
}

variable "database_password" {
  type = string
}

resource "shell_script" "run_deployment" {
  lifecycle_commands {
    create = "echo 'DATABASE_URL: ${var.database_url}' && echo 'DATABASE_PASSWORD: ${var.database_password}'"
    delete = "deleting variables"
  }
}

output "environment_info" {
  value = {
    database_url = var.database_url
    database_password = var.database_password
  }
}
