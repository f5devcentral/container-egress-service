# Container Egress Services (CES)
[![standard-readme compliant](https://img.shields.io/badge/readme%20style-standard-brightgreen.svg?style=flat-square)](https://github.com/f5devcentral/container-egress-service) [![Action Build Status](https://github.com/f5devcentral/container-egress-service/workflows/Build/badge.svg)](https://github.com/f5devcentral/container-egress-service/actions) [![Docker pull](https://img.shields.io/docker/pulls/f5devcentral/ces-controller)](https://hub.docker.com/r/f5devcentral/ces-controller) [![Issues](https://img.shields.io/github/issues/f5devcentral/container-egress-service)](https://github.com/f5devcentral/container-egress-service/issues) [![Stars](https://img.shields.io/github/stars/f5devcentral/container-egress-service)]() [![Go](https://goreportcard.com/badge/github.com/f5devcentral/container-egress-service)](https://goreportcard.com/report/github.com/f5devcentral/container-egress-service) [![License](https://img.shields.io/github/license/f5devcentral/container-egress-service)](./LICENSE) 

CES is a solution. It is used to help users manage the outgoing traffic of k8s pod/container better. It solves the challenge of outgoing traffic policy control in high dynamic IP scenarios in k8s native way, and provides a wealth of outgoing control capability. And through the hierarchical design, it solves the multi-role coordination problem among enterprise security, network, platform, and application operation departments.

## Table of Contents
- [Container Egress Services (CES)](#container-egress-services-ces)
  - [Table of Contents](#table-of-contents)
  - [Background](#background)
  - [Install](#install)
  - [Usage](#usage)
  - [Building](#building)
  - [Challenges solved](#challenges-solved)
  - [Capabilities](#capabilities)
  - [Documents](#documents)
  - [Support](#support)
  - [Community Code of Conduct](#community-code-of-conduct)
  - [Contact](#contact)
  - [License](#license)



## Background

Kubernetes is piloting projects transition to enterprise-wide application rollouts, companies must be able to extend their existing enterprise security architecture into the Kubernetes environment. There are 2 challenges here. One is technology, how  enterprise security devices to work in high dynamic IP environment. This will  introduces additional complexity and risk to traditional process. The second one is the blurry work boundary between enterprise security team, network team, platform team and application team. Security is not the responsibility of one team, it is shared. Security team/network team, platform and application team all should get its role and benefit from this shared mode. 

CES is a solution help customers to resolve the above 2 challenges. It provides k8s native way to k8s egress traffic policy tuning. Working with F5 AFM.

By running CES controller in k8s, it will automatcially create policy rules into F5 AFM. No matter IP change or scaled.

By scoped policy designment, Security/network team, platform team, application team all can participate into the policy setting. Policy management can be delegated or centralized, follow container platform's RBAC. 

<img src="https://github.com/f5devcentral/container-egress-service/wiki/img/image-20211205152836043.png" alt="scoped CRD"/>


## Install

1. Download the installation script

```
wget https://raw.githubusercontent.com/f5devcentral/container-egress-service/master/dist/install.sh
```

2. Edit the  `install.sh` script, edit the following variable values according to the actual environment. For detail, check the [wiki](https://github.com/f5devcentral/container-egress-service/wiki/2.CES%E5%AE%89%E8%A3%85)

## Usage

Please check the [Wiki](https://github.com/f5devcentral/container-egress-service/wiki) for different usages.

## Building

Docker image:
```
#GO_VERSION = 1.16
git clone https://github.com/f5devcentral/container-egress-service.git
cd container-egress-service
make release
```


## Challenges solved

- High-frequency changes in outbound traffic caused by container IP dynamics
- Different role groups have different requirements for the scope setting of the policy, and the policy needs to match the role in multiple dimensions
- Dynamic bandwidth limit requirements for outbound traffic
- Protocol in-depth security inspection requirements
- Advanced requirements for flow programmable based on access control events
- Visualization requirements for outbound traffic

## Capabilities

- Dynamic IP ACL control with Cluster/Pod/NS granularity
- Cluster/Pod/NS granular FQDN ACL control
- Time-based access control
- Matched flow event trigger and programmable
- Matched traffic redirection
- Protocol security and compliance testing
- IP intelligence
- Traffic matching log
- Traffic matching visualization report
- Protocol detection visual report
- TCP/IP Errors report
- NAT control and logging
- Data flow visualization tracking
- Visual simulation of access rules
- Transparent detection mode
- High-speed log outgoing



## Documents

Check [Release notes](https://github.com/f5devcentral/container-egress-service/releases/tag/v0.5.0).

Check the [Wiki](https://github.com/f5devcentral/container-egress-service/wiki) first.

## Support

For support, please open a GitHub issue.  Note, the code in this repository is community supported and is not supported by F5 Networks.  For a complete list of supported projects please reference [SUPPORT.md](SUPPORT.md).

## Community Code of Conduct
Please refer to the [F5 DevCentral Community Code of Conduct](code_of_conduct.md).

## Contact

j.lin@f5.com



## License

[Apache License 2.0](./LICENSE)

