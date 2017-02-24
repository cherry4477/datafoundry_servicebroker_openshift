1. 将etcd中的bsi的instance列表存到bsi-databaseshare-instances-in-etcd和bsi-openshift-instances-in-etcd中:

export ETCDCTL="etcdctl --timeout 15s --total-timeout 30s --endpoints http://xxx.yyy.zzz:2379 --username user:password"

<<<<<<< Updated upstream
$ETCDCTL ls /servicebroker/databaseshare/instance/ > bsi-databaseshare-instances-in-etcd
$ETCDCTL ls /servicebroker/openshift/instance/ > bsi-openshift-instances-in-etcd

2. 将openshift中每个区所有bsi的yaml格式信息存到region-north1-bsis等文件中
=======
1. 将etcd中的bsi的instance列表存到bsi-databaseshare-instances-in-etcd和bsi-openshift-instances-in-etcd中:

$ETCDCTL ls /servicebroker/databaseshare/instance/ > bsi-databaseshare-instances-in-etcd 2>&1

$ETCDCTL ls /servicebroker/openshift/instance/ > bsi-openshift-instances-in-etcd 2>&1

  将所有存在etcd中的instance info存入bsi-databaseshare-instances-info-in-etcd和bsi-openshift-instances-info-in-etcd中:

cat bsi-databaseshare-instances-in-etcd | while read line; do if [[ -n "${line// }" ]]; then $ETCDCTL get $line/_info; fi; done > bsi-databaseshare-instances-info-in-etcd 2>&1

cat bsi-openshift-instances-in-etcd | while read line; do if [[ -n "${line// }" ]]; then $ETCDCTL get $line/_info; fi; done > bsi-openshift-instances-info-in-etcd 2>&1

2. 将openshift中每个区所有bsi的yaml格式信息存到bsis-region-north1等文件中

# oc login region north1 (必选)
oc get bsi -o yaml --all-namespaces > bsis-region-north1 2>&1
# oc login region north2 (可选)
oc get bsi -o yaml --all-namespaces > bsis-region-north2 2>&1
>>>>>>> Stashed changes

  将openshift中每个区中所有的bsi的pod列出并存到bsi-pods-region-north1等文件中
  
# oc login region north1 (必选)
oc get pod -n service-brokers > bsi-pods-region-north1 2>&1
# oc login region north2 (可选)
oc get pod -n service-brokers > bsi-pods-region-north2 2>&1

3. 运行wild-res-finder.go查找没有bsi对应的etcd中的野instance IDs

go run wild-res-finder.go bsi-openshift-wild-instances-in-etcd > bsi-openshift-wild-instances-in-etcd
go run wild-res-finder.go bsi-databaseshare-wild-instances-in-etcd > bsi-databaseshare-wild-instances-in-etcd

4. 将野instance的info列出并写入wild-bsi-info-openshift

cat bsi-openshift-wild-instances-in-etcd | while read line; do if [[ -n "${line// }" ]]; then $ETCDCTL get /servicebroker/openshift/instance/$line/_info; fi; done > wild-bsi-info-openshift 2>&1

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

7. 运行wild-res-finder.go查找ETCD中没有记录的bsi random IDs，并存入unrecorded-bsi-random-ids

go run wild-res-finder.go unrecorded-bsi-random-ids > unrecorded-bsi-random-ids

8. 删除unrecorded bsi资源 (需略微谨慎)

# delete dc
cat unrecorded-bsi-random-ids | while read line; do if [[ -n "${line// }" ]]; then echo ====== $line && oc get dc | grep $line | cut -d' ' -f1 | while read id; do oc delete dc $id; done; fi done

# delete route
cat unrecorded-bsi-random-ids | while read line; do if [[ -n "${line// }" ]]; then echo ====== $line && oc get route | grep $line | cut -d' ' -f1 | while read id; do oc delete route $id; done; fi done

# delete svc
cat unrecorded-bsi-random-ids | while read line; do if [[ -n "${line// }" ]]; then echo ====== $line && oc get svc | grep $line | cut -d' ' -f1 | while read id; do oc delete svc $id; done; fi done

# delete rc
cat unrecorded-bsi-random-ids | while read line; do if [[ -n "${line// }" ]]; then echo ====== $line && oc get rc | grep $line | cut -d' ' -f1 | while read id; do oc delete rc $id; done; fi done

# delete pod
cat unrecorded-bsi-random-ids | while read line; do if [[ -n "${line// }" ]]; then echo ====== $line && oc get pod | grep $line | cut -d' ' -f1 | while read id; do oc delete pod $id; done; fi done

# echo pvc (be careful to delete them)
cat unrecorded-bsi-random-ids | while read line; do if [[ -n "${line// }" ]]; then echo ====== $line && oc get pvc | grep $line | cut -d' ' -f1 | while read id; do echo abc: $id; done; fi done


