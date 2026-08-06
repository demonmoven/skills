FROM python:3.11
ADD . /app
WORKDIR /app
USER root
RUN pip install -r requirements.txt
CMD ["python", "-m", "mypkg"]
