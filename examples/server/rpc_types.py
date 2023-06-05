from dataclasses import dataclass
from typing import Optional, List, Dict
from dataclasses_json import dataclass_json

class Endpoints:
    FINGERPRINT = "/fingerprint"
    STATS = "/stats"
    RESERVE = "/reserve"


@dataclass_json
@dataclass
class Attribute:
    Float: Optional[float] = None
    Int: Optional[int] = None
    String: Optional[str] = None
    Bool: Optional[bool] = None
    Unit: str = ""


@dataclass_json
@dataclass
class DeviceLocality:
    PciBusID: str


@dataclass_json
@dataclass
class Device:
    ID: str
    Healthy: bool
    HealthDesc: str
    HwLocality: Optional[DeviceLocality]


@dataclass_json
@dataclass
class DeviceGroup:
    Vendor: str
    Type: str
    Name: str
    Devices: List[Device]
    Attributes: Dict[str, Attribute]


@dataclass_json
@dataclass
class FingerprintResponse:
    Devices: List[DeviceGroup]
    Error: Optional[str] = None


@dataclass_json
@dataclass
class Mount:
    TaskPath: str
    HostPath: str
    ReadOnly: bool


@dataclass_json
@dataclass
class DeviceSpec:
    TaskPath: str
    HostPath: str
    CgroupPerms: str


@dataclass_json
@dataclass
class ContainerReservation:
    Envs: Dict[str, str]
    Mounts: List[Mount]
    Devices: List[DeviceSpec]


@dataclass_json
@dataclass
class StatValue:
    FloatNumeratorVal: Optional[float] = None
    FloatDenominatorVal: Optional[float] = None
    IntNumeratorVal: Optional[int] = None
    IntDenominatorVal: Optional[int] = None
    StringVal: Optional[str] = None
    BoolVal: Optional[bool] = None
    Unit: str = ""
    Desc: str = ""


@dataclass_json
@dataclass
class StatObject:
    Nested: Dict[str, "StatObject"]
    Attributes: Dict[str, StatValue]


@dataclass_json
@dataclass
class DeviceStats:
    Summary: Optional[StatValue] = None
    Stats: Optional[StatObject] = None
    Timestamp: Optional[str] = None


@dataclass_json
@dataclass
class DeviceGroupStats:
    Vendor: str
    Type: str
    Name: str
    InstanceStats: Dict[str, DeviceStats]


@dataclass_json
@dataclass
class StatsResponse:
    Groups: List[DeviceGroupStats]
    Error: Optional[str] = None


@dataclass_json
@dataclass
class PluginConfig:
    Address: str = "http://127.0.0.1:5656/"
    FingerprintPeriod: str = "1m"
