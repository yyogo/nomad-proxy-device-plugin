import datetime
import random
from dataclasses import asdict

from flask import Flask, abort, jsonify, request

from .rpc_types import *

app = Flask(__name__)


@app.route(Endpoints.FINGERPRINT)
def fingerprint():
    return jsonify(
        asdict(FingerprintResponse(
            Devices=[
                DeviceGroup(
                    Vendor="Test",
                    Type="Hello",
                    Name="World",
                    Devices=[
                        Device(
                            ID="1234", Healthy=True, HealthDesc="OK", HwLocality=None
                        )
                    ],
                    Attributes={"GenericAttr": Attribute(String="Yee")},
                )
            ]
        ))
    )


@app.route(Endpoints.RESERVE, methods=['POST'])
def reserve():
    device_ids = request.json
    if not isinstance(device_ids, list) or not all(isinstance(x, str) for x in device_ids):
        abort(413)
    return jsonify(ContainerReservation({i: "OK" for i in device_ids}, [], []))


@app.route(Endpoints.STATS)
def stats():
    return jsonify(
        StatsResponse(
            Groups=[
                DeviceGroupStats(
                    Vendor="Test",
                    Type="Hello",
                    Name="World",
                    InstanceStats={
                        "1234": DeviceStats(
                            Summary=StatValue(StringVal="Coolio"),
                            Stats=StatObject(
                                Nested={},
                                Attributes={
                                    "random": StatValue(
                                        FloatNumeratorVal=random.random()
                                    ),
                                },
                            ),
                            Timestamp=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                        ),
                    },
                )
            ]
        )
    )


if __name__ == "__main__":
    app.run(debug=True, port=5656)
