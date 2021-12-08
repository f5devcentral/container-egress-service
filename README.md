## Container Egress Services (CES)
Kubernetes is piloting projects transition to enterprise-wide application rollouts, companies must be able to extend their existing enterprise security architecture into the Kubernetes environment. There are 2 challenges here. One is technology, how  enterprise security devices to work in high dynamic IP environment. This will  introduces additional complexity and risk to traditional process. The second one is the blurry work boundary between enterprise security team, network team, platform team and application team. Security is not the responsibility of one team, it is shared. Security team/network team, platform and application team all should get its role and benefit from this shared mode. 

CES is a solution help customers to resolve the above 2 challenges. It provides k8s native way to k8s egress traffic policy tuning. Working with F5 AFM.



By running CES controller in k8s, it will automatcially create policy rules into F5 AFM. No matter IP change or scaled.

By scoped policy designment, Security/network team, platform team, application team all can participate into the policy setting. Policy management can be delegated or centralized, follow container platform's RBAC. 

<img src="https://github.com/f5devcentral/container-egress-service/wiki/img/image-20211205152836043.png" alt="scoped CRD"/>



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



## Contact

j.lin@f5.com



