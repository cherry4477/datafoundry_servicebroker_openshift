1. 将etcd中的bsi的instance列表存到bsi-databaseshare-instances-in-etcd和bsi-openshift-instances-in-etcd中:

export ETCDCTL="etcdctl --timeout 15s --total-timeout 30s --endpoints http://xxx.yyy.zzz:2379 --username user:password"

$ETCDCTL ls /servicebroker/databaseshare/instance/ > bsi-databaseshare-instances-in-etcd
$ETCDCTL ls /servicebroker/openshift/instance/ > bsi-openshift-instances-in-etcd

2. 将openshift中每个区所有bsi的yaml格式信息存到region-north1-bsis等文件中

# oc login region north1 (必选)
oc get bsi -o yaml --all-namespaces > region-north1-bsis
# oc login region north2 (可选)
oc get bsi -o yaml --all-namespaces > region-north2-bsis

3. 运行wild-res-finder.go查找没有bsi对应的etcd中的野instance IDs

go run wild-res-finder.go bsi-openshift-wild-instances-in-etcd > bsi-openshift-wild-instances-in-etcd
go run wild-res-finder.go bsi-databaseshare-wild-instances-in-etcd > bsi-databaseshare-wild-instances-in-etcd

4. 将野instance的info列出并写入wild-bsi-info-openshift

cat bsi-openshift-wild-instances-in-etcd | while read line; do if [[ -n "${line// }" ]]; then echo $ETCDCTL get /servicebroker/openshift/instance/$line/_info; fi; done > wild-bsi-info-openshift

5. 运行wild-res-finder.go查找野bsi资源random IDs，并存入wild-bsi-random-ids

go run wild-res-finder.go wild-bsi-random-ids > wild-bsi-random-ids

6. 删除野bsi资源

# delete dc
cat wild-bsi-random-ids | while read line; do if [[ -n "${line// }" ]]; then echo ====== $line && oc get dc | grep $line | cut -d' ' -f1 | while read id; do oc delete dc $id; done; fi done

# delete route
cat wild-bsi-random-ids | while read line; do if [[ -n "${line// }" ]]; then echo ====== $line && oc get route | grep $line | cut -d' ' -f1 | while read id; do oc delete route $id; done; fi done

# delete svc
cat wild-bsi-random-ids | while read line; do if [[ -n "${line// }" ]]; then echo ====== $line && oc get svc | grep $line | cut -d' ' -f1 | while read id; do oc delete svc $id; done; fi done

# delete rc
cat wild-bsi-random-ids | while read line; do if [[ -n "${line// }" ]]; then echo ====== $line && oc get rc | grep $line | cut -d' ' -f1 | while read id; do oc delete rc $id; done; fi done

# delete pod
cat wild-bsi-random-ids | while read line; do if [[ -n "${line// }" ]]; then echo ====== $line && oc get pod | grep $line | cut -d' ' -f1 | while read id; do oc delete pod $id; done; fi done

# echo pvc (be careful to delete them)
cat wild-bsi-random-ids | while read line; do if [[ -n "${line// }" ]]; then echo ====== $line && oc get pvc | grep $line | cut -d' ' -f1 | while read id; do echo abc: $id; done; fi done


