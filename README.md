# datafoundry_servicebroker_openshift

need golang 1.6+ to build and run

# about instance id

bsi的实例id都通过NewThirteenLengthID生成。

目前，因为
for limited fields stored in ETCD, now 

# install

oc login ...
oc project servicebrokers-openshift
oc new-build https://github.com/asiainfoLDP/etcd-openshift-orchestration 
oc new-build https://github.com/asiainfoLDP/zookeeper-openshift-orchestration --context-dir='image'

oc new-app --name servicebroker-openshift https://github.com/asiainfoLDP/datafoundry_servicebroker_openshift#develop \
    -e  ETCDENDPOINT="..."  \
    -e  ETCDUSER="..." \
    -e  ETCDPASSWORD="..." \
    -e  BROKERPORT="8888"  \
    -e  OPENSHIFTADDR="..."  \
    -e  OPENSHIFTUSER="...."   \
    -e  OPENSHIFTPASS="..."  \
    -e  SBNAMESPACE="servicebrokers-openshift"   \
    -e  ETCDIMAGE="servicebrokers-openshift/etcd-openshift-orchestration"   \
    -e  ZOOKEEPERIMAGE="servicebrokers-openshift/zookeeper-openshift-orchestration"   \
    -e  ENDPOINTSUFFIX="..."

ENDPOINTSUFFIX: the suffix of routes

