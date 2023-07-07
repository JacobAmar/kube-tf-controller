import subprocess, os,re, json
def run_command(command):
    process = subprocess.run(command,universal_newlines=True,capture_output=True)
    try:
       process.check_returncode()
       return process.stdout
    except Exception:
        print("terraform command has failed, exiting")
        print(process.stderr)

def terraform_init():
    process = run_command(["terraform","init","-input=false","-no-color"])
    print(process)

def terraform_plan():
    process = run_command(["terraform","plan","-input=false","-no-color","-out=tfplan"])
    # using regex to capture terraform changes
    print(process)
    result = re.search("Plan:\s(\d).*(\d)\sto\schange.*(\d)\sto\sdestroy",process)
    if result != None:
      to_add = result.group(1)
      to_change = result.group(2)
      to_destroy = result.group(3)
      # only apply if add / change / destroy is != to 0
      if to_add != 0 or to_change !=0 or to_destroy != 0:
          return True
      else:
          return False
    else:
        # check if theres any changes to outputs:
        result = re.search("Changes to Outputs:",process)
        if result != None:
            return True
        else:
            return False


def terraform_apply():
    process = run_command(["terraform","apply","-input=false","-no-color","tfplan"])
    print(process)

def tf_run(path,variables=None):
    os.environ["TF_IN_AUTOMATION"] = "true"
    os.chdir(path)
    if variables != None:
        variables_json = json.dumps(variables)
        with open("tfoperator.auto.tfvars.json", "w") as file:
             file.write(variables_json)
    terraform_init()
    plan = terraform_plan()
    if plan:
        terraform_apply()
    else:
        print("No changes to apply")

