import time
from locust import FastHttpUser, task, between
from uuid import uuid4


class FeederUser(FastHttpUser):
    sync_code = None
    user_id = None
    feeds_etag = None
    devices_etag = None
    feeds_etag = ""
    wait_time = between(1, 5)

    @task
    def get_devices(self):
        with self.rest("GET",
            "/api/v1/devices",
            headers={
                "Content-Type": "application/json",
                "Authorization": "Basic ZmVlZGVyX3VzZXI6ZmVlZGVyX3NlY3JldF8xMjM0",
                "X-FEEDER-ID": self.sync_code,
                #"X-FEEDER-USER-ID": self.user_id,
                "X-FEEDER-DEVICE-ID": self.device_id
            }
        ) as response:
            pass

    @task
    def get_feeds(self):
        with self.rest("GET",
            "/api/v1/feeds",
            headers={
                "Content-Type": "application/json",
                "Authorization": "Basic ZmVlZGVyX3VzZXI6ZmVlZGVyX3NlY3JldF8xMjM0",
                "X-FEEDER-ID": self.sync_code,
                #"X-FEEDER-USER-ID": self.user_id,
                "X-FEEDER-DEVICE-ID": self.device_id
            }
        ) as response:
            pass

    @task
    def get_ereadmark(self):
        with self.rest("GET",
            "/api/v1/ereadmark",
            headers={
                "Content-Type": "application/json",
                "Authorization": "Basic ZmVlZGVyX3VzZXI6ZmVlZGVyX3NlY3JldF8xMjM0",
                "X-FEEDER-ID": self.sync_code,
                #"X-FEEDER-USER-ID": self.user_id,
                "X-FEEDER-DEVICE-ID": self.device_id
            }
        ) as response:
            pass

    @task
    def post_feeds(self):
        with self.rest("POST",
            "/api/v1/feeds",
            json={
                "contentHash": 123456789,
                "encrypted": "encrypted_feed_content"
            },
            headers={
                "Content-Type": "application/json",
                "Authorization": "Basic ZmVlZGVyX3VzZXI6ZmVlZGVyX3NlY3JldF8xMjM0",
                "X-FEEDER-ID": self.sync_code,
                #"X-FEEDER-USER-ID": self.user_id,
                "X-FEEDER-DEVICE-ID": self.device_id,
                "If-Match": self.feeds_etag
            },
        ) as response:
            self.feeds_etag = response.headers["ETag"]

    @task(10)
    def post_ereadmark(self):
        with self.rest("POST",
            "/api/v1/ereadmark",
            json={
                "items": [
                    {
                        "encrypted": str(uuid4()),
                    }
                ]
            },
            headers={
                "Content-Type": "application/json",
                "Authorization": "Basic ZmVlZGVyX3VzZXI6ZmVlZGVyX3NlY3JldF8xMjM0",
                "X-FEEDER-ID": self.sync_code,
                #"X-FEEDER-USER-ID": self.user_id,
                "X-FEEDER-DEVICE-ID": self.device_id
            }
        ) as response:
            pass

    def on_start(self):
        with self.rest("POST",
            "/api/v1/create",
            json={
                "deviceName": "alice"
            },
            headers={
                "Content-Type": "application/json",
                "Authorization": "Basic ZmVlZGVyX3VzZXI6ZmVlZGVyX3NlY3JldF8xMjM0",
            },
        ) as response:
            result = response.js
            self.device_id = result["deviceId"]
            #self.user_id = result["userId"]
            self.sync_code = result["syncCode"]
            #raise ValueError("User ID: %s, Device ID: %s" % (self.user_id, self.device_id))
            #raise ValueError("Sync Code: %s, Device ID: %s" % (self.sync_code, self.device_id))

    def on_stop(self):
        with self.rest("DELETE",
            f"/api/v1/devices/{self.device_id}",
            headers={
                "Content-Type": "application/json",
                "Authorization": "Basic ZmVlZGVyX3VzZXI6ZmVlZGVyX3NlY3JldF8xMjM0",
                "X-FEEDER-ID": self.sync_code,
                "X-FEEDER-DEVICE-ID": self.device_id,
            },
        ) as response:
            pass

