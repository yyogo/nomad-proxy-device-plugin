Nomad REST Proxy Device Plugin
==============================

This [Nomad device plugin](https://www.nomadproject.io/docs/internals/plugins/devices.html) 
is a simple wrapper that calls an external REST API to obtain the device information.

Based on the [Nomad skeleton device plugin](https://github.com/hashicorp/nomad-skeleton-device-plugin).
This is intended for devices for which the support framework is already written in another language other than Go.

A simple server implementation in Python is provided in `examples/server`.

Check readme in the original repo for more information.
