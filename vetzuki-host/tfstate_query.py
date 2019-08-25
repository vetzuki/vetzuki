#!/usr/bin/python3.7
import json
import sys

TFSTATE = "terraform.tfstate"

def find_resource(resources, resource_type, name, prop):
    of_type = list(
        filter(
            lambda r: r["type"] == resource_type and r["name"] == name,
            resources))
    if len(of_type) == 0:
        return None

    values = []
    for resource in of_type:
        for instance in resource["instances"]:
            try:
                values.append(instance["attributes"][prop])
            except KeyError:
                pass
    return ",".join(values)

def main():
    query = sys.argv[1]
    resource,name,prop = query.split('.')

    with open("terraform.tfstate") as tfstate:
        j = json.load(tfstate)
        value = find_resource(j["resources"], resource, name, prop)
        if value:
            print("{}={}".format(query, value))

if __name__ == "__main__":
          main()
