from kubernetes import client, config , watch
import logging

logging.basicConfig(level=logging.INFO)
# Configs can be set in Configuration class directly or using helper utility
config.load_kube_config()
v1 = client.CoreV1Api()
crd = client.CustomObjectsApi()
#crd.list_cluster_custom_object('terraform.iac.operator','v1','tfprojects')
watch = watch.Watch()
resource_version = ""
for event in watch.stream(crd.list_cluster_custom_object,'terraform.iac.operator','v1','tfbackends',resource_version=resource_version):
    repository = event["object"]["spec"]["repository"]
    terraform_directory = event["object"]["spec"]["path"]
    terraform_variables = event["object"]["spec"]["variables"]
    project_name = event["object"]["metadata"]["name"]
    repo_directory = "cloned"
    event_type = event["type"]
    resource_version = event["object"]["metadata"]["resourceVersion"]
    if event_type == "ADDED" or event_type == "MODIFIED":
        