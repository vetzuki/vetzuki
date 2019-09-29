# GetProspect: Lambda
resource "aws_lambda_function" "getProspectLambda" {
    filename = "getProspect/lambda.zip"
    function_name = "${var.getProspectLambdaName}"
    role = "${aws_iam_role.lambdaRole.arn}"
    handler = "handler"
    runtime = "go1.x"
    source_code_hash = "${filebase64sha256("getProspect/lambda.zip")}"
    timeout = 10
    vpc_config {
      subnet_ids = [
        "subnet-028bcbeafe9d69115",
        "subnet-031837afae45ee3b8"
      ]
      security_group_ids = ["sg-013a87f09a9a2cf14"]
    }
    environment {
      variables = {
        LDAP_HOST = "172.16.0.100:389"
        BASE_DN = "ou=prospects,dc=poc,dc=vetzuki,dc=com"
        GROUPS_DN = "ou=groups,dc=poc,dc=vetzuki,dc=com"
        BIND_DN = "cn=admin,dc=poc,dc=vetzuki,dc=com"
        BIND_PASSWORD = "vetzuk1p0c"
        VETZUKI_ENVIRONMENT = "poc"
        REDIS_HOST = "172.16.0.100:6379"
        REDIS_PASSWORD = ""
        REDIS_DB = ""
        SSH_URL = "ssh.${var.vetzukiEnvironment}.vetzuki.com"
        DB_HOST = "db.${var.vetzukiEnvironment}.vetzuki.com"
        DB_PORT = "${var.db_port}"
        DB_USERNAME = "${var.db_user}"
        DB_PASSWORD = "${var.db_password}"
        DB_NAME = "${var.db_name}"

      }
    }
}
# CreateProspect: Lambda
resource "aws_lambda_function" "createProspect" {
    filename = "createProspect/lambda.zip"
    function_name = "createProspect-${var.vetzukiEnvironment}"
    role = "${aws_iam_role.lambdaRole.arn}"
    handler = "handler"
    runtime = "go1.x"
    source_code_hash = "${filebase64sha256("createProspect/lambda.zip")}"
    timeout = 10
    vpc_config {
      subnet_ids = [
        "subnet-028bcbeafe9d69115",
        "subnet-031837afae45ee3b8"
      ]
      security_group_ids = ["sg-013a87f09a9a2cf14"]
    }
    environment {
      variables = {
        LDAP_HOST = "172.16.0.100:389"
        BASE_DN = "ou=prospects,dc=poc,dc=vetzuki,dc=com"
        GROUPS_DN = "ou=groups,dc=poc,dc=vetzuki,dc=com"
        BIND_DN = "cn=admin,dc=poc,dc=vetzuki,dc=com"
        BIND_PASSWORD = "vetzuk1p0c"
        VETZUKI_ENVIRONMENT = "poc"
        REDIS_HOST = "172.16.0.100:6379"
        REDIS_PASSWORD = ""
        REDIS_DB = ""
        SSH_URL = "ssh.${var.vetzukiEnvironment}.vetzuki.com"
        DB_HOST = "db.${var.vetzukiEnvironment}.vetzuki.com"
        DB_PORT = "${var.db_port}"
        DB_USERNAME = "${var.db_user}"
        DB_PASSWORD = "${var.db_password}"
        DB_NAME = "${var.db_name}"
      }
    }
}

# CreateProspect : Allow APIGateway to execute CreateProspect Lambda
resource "aws_lambda_permission" "createProspect" {
    statement_id = "AllowExecutionFromAPIGateway"
    action = "lambda:InvokeFunction"
    function_name = "${aws_lambda_function.createProspect.function_name}"
    principal = "apigateway.amazonaws.com"
}