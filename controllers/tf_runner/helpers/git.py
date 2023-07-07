import os,subprocess

def git_handler(repository,path,branch=None):
            # Perform the git clone operation if the directory doesn't exist
    if not os.path.isdir(path):
        subprocess.run(["git", "clone", repository, path])
    if branch != None:
    # Checkout to the specific version
       subprocess.run(["git", "checkout", branch], cwd=path)
       subprocess.run(["git", "pull"])
    else:
        subprocess.run(["git", "pull"],cwd=path)