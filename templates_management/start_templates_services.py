import subprocess

# Number of replicas per service
num_replicas = 3

# Start each replica on a separate port
for port in range(5005, 5005 + num_replicas):
    subprocess.Popen(["/opt/homebrew/bin/python3", "templates_service.py", str(port)])