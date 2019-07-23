provider "aws" {
  profile = "terraformPOC"
  region = "us-west-2"
}

resource "aws_vpc" "poc_vpc" {
  cidr_block = "172.16.0.0/24"
  tags = {
    Environment = "poc"
    Name = "poc"
  }
}

resource "aws_subnet" "poc_subnet" {
  vpc_id = "${aws_vpc.poc_vpc.id}"
  cidr_block = "172.16.0.0/24"
  availability_zone = "us-west-2a"
  tags = {
    Environment = "poc"
    Name = "poc"
  }
}

resource "aws_network_interface" "poc_exam_host" {
  subnet_id = "${aws_subnet.poc_subnet.id}"
  private_ips = ["172.16.0.100"]
  security_groups = ["${aws_security_group.poc_ssh.id}"]
  tags = {
    Environment = "poc"
    Name = "poc"
  }
  depends_on = ["aws_security_group.poc_ssh"]
}
resource "aws_internet_gateway" "poc" {
  vpc_id = "${aws_vpc.poc_vpc.id}"
}
resource "aws_route" "poc_default_route" {
  route_table_id = "rtb-02838a73b97aedf24"
  destination_cidr_block = "0.0.0.0/0"
  gateway_id = "${aws_internet_gateway.poc.id}"
}
resource "aws_eip" "poc" {
  instance = "${aws_instance.poc_exam_host.id}"
  associate_with_private_ip = "172.16.0.100"
  vpc = true
  depends_on = ["aws_internet_gateway.poc"]
}

resource "aws_security_group" "poc_ssh" {
  name = "POC"
  description = "Allow SSH POC traffic"
  vpc_id = "${aws_vpc.poc_vpc.id}"
  ingress {
    from_port = 22
    to_port = 22
    protocol = "tcp"
    cidr_blocks = ["71.204.152.97/32"]
  }
  egress {
    from_port = 0
    to_port = 0
    protocol = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
  tags = {
    Environment = "poc"
    Name = "poc"
  }
}
    

resource "aws_instance" "poc_exam_host" {
  ami = "ami-082b5a644766e0e6f"
  instance_type = "t3.micro"

  network_interface {
    network_interface_id = "${aws_network_interface.poc_exam_host.id}"
    device_index = 0
  }
  
  tags = {
    Environment = "poc"
    Name = "poc"
  }
}
