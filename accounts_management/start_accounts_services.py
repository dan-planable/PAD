import subprocess

# Number of replicas per service
num_replicas = 3

# Start each replica on a separate port
for port in range(5000, 5000 + num_replicas):
    subprocess.Popen(["/opt/homebrew/bin/python3", "accounts_service.py", str(port)])
