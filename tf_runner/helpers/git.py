from git import Repo, Head
import os


def git_handler(repository,path,branch=None):
        if os.path.exists(path) == False:
           print(f"cloning the repository: {repository}")
           Repo.clone_from(repository,to_path=path,branch=branch)
        else:
                repo = Repo(path)
                remote = repo.remote(name='origin')
                if branch == None:
                    print(f"pulling changes for repo {repository}")
                    pull = remote.pull()
                    for change in pull:
                          print(f"pulled changes for {change.name}")
                else:
                      # checking out to the spesific branch address any changes on the branch name
                      print(f"pulling changes for repo {repository}")
                      for ref in remote.refs:
                            if branch in ref.name:
                                  # meaning, if the branch name is in the remote
                                  print(f"branch found, {ref.name}")
                                  remote.fetch()
                                  ref.checkout()
                                  #remote.pull()