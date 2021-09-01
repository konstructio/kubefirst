resource "aws_appautoscaling_target" "dynamodb-table-read-target" {
  max_capacity       = 100
  min_capacity       = 10
  resource_id        = "table/${aws_dynamodb_table.vault_dynamodb_table.name}"
  scalable_dimension = "dynamodb:table:ReadCapacityUnits"
  service_namespace  = "dynamodb"
}

resource "aws_appautoscaling_policy" "dynamodb-table-read-policy" {
  name               = "DynamoDBReadCapacityUtilization:${aws_appautoscaling_target.dynamodb-table-read-target.resource_id}"
  policy_type        = "TargetTrackingScaling"
  resource_id        = aws_appautoscaling_target.dynamodb-table-read-target.resource_id
  scalable_dimension = aws_appautoscaling_target.dynamodb-table-read-target.scalable_dimension
  service_namespace  = aws_appautoscaling_target.dynamodb-table-read-target.service_namespace

  target_tracking_scaling_policy_configuration {
    predefined_metric_specification {
      predefined_metric_type = "DynamoDBReadCapacityUtilization"
    }

    target_value = 30
  }
}

resource "aws_appautoscaling_target" "dynamodb-table-write-target" {
  max_capacity       = 100
  min_capacity       = 10
  resource_id        = "table/${aws_dynamodb_table.vault_dynamodb_table.name}"
  scalable_dimension = "dynamodb:table:WriteCapacityUnits"
  service_namespace  = "dynamodb"
}

resource "aws_appautoscaling_policy" "dynamodb_table_write_policy" {
  name               = "DynamoDBWriteCapacityUtilization:${aws_appautoscaling_target.dynamodb-table-write-target.resource_id}"
  policy_type        = "TargetTrackingScaling"
  resource_id        = aws_appautoscaling_target.dynamodb-table-write-target.resource_id
  scalable_dimension = aws_appautoscaling_target.dynamodb-table-write-target.scalable_dimension
  service_namespace  = aws_appautoscaling_target.dynamodb-table-write-target.service_namespace

  target_tracking_scaling_policy_configuration {
    predefined_metric_specification {
      predefined_metric_type = "DynamoDBWriteCapacityUtilization"
    }

    target_value = 30
  }
}

resource "aws_dynamodb_table" "vault_dynamodb_table" {
  # todo https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/dynamodb_table#point_in_time_recovery
  name           = "vault-dynamodb-backend"
  billing_mode   = "PROVISIONED"
  read_capacity  = 15
  write_capacity = 33
  hash_key       = "Path"
  range_key      = "Key"
  stream_enabled = false
  point_in_time_recovery {
    enabled = true
  }

  attribute {
    name = "Path"
    type = "S"
  }

  attribute {
    name = "Key"
    type = "S"
  }

  tags = {
    Name          = "vault-dynamodb-backend"
    VaultInstance = "vault"
    Environment   = "mgmt"
  }

  lifecycle {
    ignore_changes = [
      read_capacity, write_capacity
    ]
  }
}
