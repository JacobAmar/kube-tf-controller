from kubernetes import client, config , watch
from git import Repo
import json
import logging
from helpers.git import git_handler
logging.basicConfig(level=logging.INFO)
# Configs can be set in Configuration class directly or using helper utility
config.load_kube_config()
v1 = client.CoreV1Api()
crd = client.CustomObjectsApi()
#crd.list_cluster_custom_object('terraform.iac.operator','v1','tfprojects')
watch = watch.Watch()
for event in watch.stream(crd.list_cluster_custom_object,'terraform.iac.operator','v1','tfprojects'):
    repository = event["object"]["spec"]["repository"]
    project_name = event["object"]["metadata"]["name"]
    repo_directory = "cloned"
    event_type = event["type"]
    if event_type == "ADDED":
        git_handler(repository=repository,path=project_name,branch="init")
        print("done")