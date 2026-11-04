# Build with the pinned upstream checkout as context. Source and MIT license
# remain upstream; this packaging disables its debugger/reloader for the lab.
FROM python:3.11-slim@sha256:db20d8506e2f15b6db39b40b6015ca46bae855dec6dd326d462cfe79ae972893
WORKDIR /vampi
COPY requirements.txt /tmp/requirements.txt
RUN pip install --no-cache-dir -r /tmp/requirements.txt Werkzeug==2.2.3
COPY . /vampi
ENV PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1
USER 10001:10001
EXPOSE 5000
ENTRYPOINT ["python", "-c", "from config import vuln_app; vuln_app.run(host='0.0.0.0', port=5000, debug=False, use_reloader=False)"]
