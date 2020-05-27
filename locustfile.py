from locust import HttpLocust, TaskSet, between, task
import uuid

class MyTaskSet(TaskSet):
    # @task(1)
    # def index(self):
    #     self.client.get("/")

    @task(50)
    def dispatch_Motorsport_Success(self):
        rand = uuid.uuid4()
        self.client.get(
            url="/dispatch?customer=123&nonse=" + str(rand),
            headers={
                "location": "VA",
                "status": "processed"
            },
        )

    @task(10)
    def dispatch_Motorsport_Failed(self):
        rand = uuid.uuid4()
        self.client.get(
            url="/dispatch?customer=123&nonse=" + str(rand),
            headers={
                "location": "VA",
                "status": "error"
            },
        )

    @task(9)
    def dispatch_Esports_Success(self):
        rand = uuid.uuid4()
        self.client.get(
            url="/dispatch?customer=392&nonse=" + str(rand),
            headers={
                "location": "MD",
                "status": "processed"
            },
        )

    @task(1)
    def dispatch_Esports_Failed(self):
        rand = uuid.uuid4()
        self.client.get(
            url="/dispatch?customer=392&nonse=" + str(rand),
            headers={
                "location": "MD",
                "status": "error"
            },
        )

    @task(12)
    def dispatch_Taxidermy_Success(self):
        rand = uuid.uuid4()
        self.client.get(
            url="/dispatch?customer=731&nonse=" + str(rand),
            headers={
                "location": "AL",
                "status": "processed"
            },
        )

    @task(3)
    def dispatch_Taxidermy_Failed(self):
        rand = uuid.uuid4()
        self.client.get(
            url="/dispatch?customer=731&nonse=" + str(rand),
            headers={
                "location": "AL",
                "status": "error"
            },
        )

    @task(13)
    def dispatch_Distillery_Success(self):
        rand = uuid.uuid4()
        self.client.get(
            url="/dispatch?customer=567&nonse=" + str(rand),
            headers={
                "location": "MA",
                "status": "processed"
            },
        )

    @task(2)
    def dispatch_Distillery_Failed(self):
        rand = uuid.uuid4()
        self.client.get(
            url="/dispatch?customer=567&nonse=" + str(rand),
            headers={
                "location": "MA",
                "status": "error"
            },
        )

class WebsiteUser(HttpLocust):
    task_set = MyTaskSet
    wait_time = between(1.0, 3.0)
