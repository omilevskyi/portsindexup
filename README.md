<!-- markdownlint-disable MD013 -->
# portsindexup

Update the FreeBSD ports INDEX file partially.

Usually when updating ports from the git repository, there is no INDEX file.
Creating an INDEX file takes significant time and CPU load.
If the INDEX file is obtained once, then it can be updated quickly and partially,
depending on what has been updated since the last time.
It is suggested to run "make index" on a daily basis, and "portsindexup" after each "git pull".

TODO: create a number of make runs equal to the number of CPUs.
