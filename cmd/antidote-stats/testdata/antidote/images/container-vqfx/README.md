# container-vQFX

This project aims to run vQFX10K with the cosim based PFE in a Docker Container, similar to https://github.com/juniper/OpenJNPR-Container-vMX, so it can be used jointly to build up datacenter topologies via docker-compose. Cosim (PFE) runs natively in the Docker container. No need for a PFE Virtual Machine!

Requires file cosim.tgz in src directory and a valid junos vqfx qcow2 image in the root directory:

```
$ ls -l *img src/cosim.tgz *tgz
-rw-r--r-- 1 mwiget mwiget 549584896 Mar 11 16:24 jinstall-vqfx-10-f-18.4R1.8.img
-rw-r--r-- 1 mwiget mwiget 570097664 Mar 26 19:59 jinstall-vqfx-10-f-18.1R1.9.img
-rw-r--r-- 1 mwiget mwiget 134064658 Mar 11 16:22 src/cosim.tgz
-rw-r--r-- 1 mwiget mwiget   3142047 Mar 26 19:31 junos-openconfig-0.0.0.10-1-signed.tgz
```

Then build the image by running ./build.sh:

```
$ ./build.sh
Sending build context to Docker daemon  527.1MB
Step 1/11 : FROM ubuntu:18.04 as cosim
 ---> 47b19964fb50
Step 2/11 : ADD cosim.tgz /root/pecosim
 ---> Using cache
 ---> a46c5fdcf5be
Step 3/11 : RUN rm -f /root/pecosim/*.tgz
 ---> Using cache
 ---> bc8b86685b99
Step 4/11 : FROM ubuntu:18.04
 ---> 47b19964fb50
Step 5/11 : RUN export DEBIAN_FRONTEND=noninteractive     && apt-get update && apt-get install -y -q qemu-kvm qemu-utils dosfstools pwgen         ca-certificates netbase libpcap0.8         tcpdump macchanger gettext-base net-tools ethtool        file iproute2 docker.io         --no-install-recommends     && mv /usr/sbin/tcpdump /sbin/     && mkdir /root/pecosim
 ---> Using cache
 ---> f6fabd89af25
Step 6/11 : COPY --from=cosim /root/pecosim /root/pecosim
 ---> Using cache
 ---> e5ef4a8f3ad2
Step 7/11 : COPY create_config_drive.sh launch.sh   create_apply_group.sh fix_network_order.sh /
 ---> Using cache
 ---> b2330ee61cd5
Step 8/11 : RUN chmod a+rx /*.sh
 ---> Using cache
 ---> 43c49893b55f
Step 9/11 : EXPOSE 22
 ---> Using cache
 ---> b7c9d20b7941
Step 10/11 : EXPOSE 830
 ---> Using cache
 ---> 6e1b16be7b33
Step 11/11 : ENTRYPOINT ["/launch.sh"]
 ---> Using cache
 ---> 7170ee88f43e
Successfully built ccf3a561f40c
Successfully tagged container-vqfx:latest
Building container antidote/vqfx:18.1R1.9 ... 
Sending build context to Docker daemon   1.65GB
Step 1/4 : FROM container-vqfx
 ---> ccf3a561f40c
Step 2/4 : ARG image
 ---> Using cache
 ---> f0efae4e82d9
Step 3/4 : ARG ocpkg
 ---> Using cache
 ---> f6a626e29968
Step 4/4 : COPY $image $ocpkg /u/
 ---> Using cache
 ---> b2459ee98508
Successfully built b2459ee98508
Successfully tagged antidote/vqfx:18.1R1.9
Building container antidote/vqfx:18.4R1.8 ... 
Sending build context to Docker daemon   1.65GB
Step 1/4 : FROM container-vqfx
 ---> ccf3a561f40c
Step 2/4 : ARG image
 ---> Using cache
 ---> f0efae4e82d9
Step 3/4 : ARG ocpkg
 ---> Using cache
 ---> f6a626e29968
Step 4/4 : COPY $image $ocpkg /u/
 ---> Using cache
 ---> b746283db2e4
Successfully built b746283db2e4
Successfully tagged antidote/vqfx:18.4R1.8
```

List images:

```
$ docker images |head
REPOSITORY                                                   TAG                 IMAGE ID            CREATED             SIZE
antidote/vqfx                                                18.4R1.8            b746283db2e4        19 minutes ago      1.28GB
antidote/vqfx                                                18.1R1.9            b2459ee98508        19 minutes ago      1.3GB
container-vqfx                                               latest              ccf3a561f40c        19 minutes ago      723MB
<none>                                                       <none>              caa698cbe391        24 minutes ago      1.28GB
```

Run it without attached network interfaces to test the user account (antidote/antidotepassword):

```
$ docker run -ti --rm --name vqfx --privileged antidote/vqfx:18.4R1.8
```

As this runs on the console, hit enter after it booted ,then log in:

```
$ docker run -ti --rm --name vqfx --privileged antidote/vqfx:18.4R1.8
Juniper Networks vQFX Docker Light Container
/u contains the following files:
Dockerfile	  README.md			   src
Dockerfile.junos  build.sh			   vagrant
Makefile	  jinstall-vqfx-10-f-18.4R1.8.img  vqfx1.yml
using qcow2 image jinstall-vqfx-10-f-18.4R1.8.img
88: eth0@if89: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue state UP mode DEFAULT group default
    link/ether 02:42:ac:11:00:02 brd ff:ff:ff:ff:ff:ff link-netnsid 0
no access to /var/run/docker.sock
Interface  IPv6 address
Bridging  with em0
Current MAC:   02:42:ac:11:00:02 (unknown)
Permanent MAC: 00:00:00:00:00:00 (XEROX CORPORATION)
New MAC:       18:aa:45:45:95:80 (Fon Technology)
ls: cannot access 'id_*.pub': No such file or directory
WARNING: Can't read ssh public key file . Creating user 'antidote' with same password as root
default route:
mygw=
-----------------------------------------------------------------------
cb3707b8151e (172.17.0.2) 18.4R1.8 root password antidotepassword
-----------------------------------------------------------------------

. . . 

.
Mon Mar 11 17:02:19 UTC 2019
Mar 11 17:02:25 init: shm-rtsdbd (PID 1974) started

cb3707b8151e (ttyd0)

login: antidote
Password:

--- JUNOS 18.4R1.8 built 2018-12-17 03:30:15 UTC
{master:0}
antidote@cb3707b8151e> show interfaces terse em0
Interface               Admin Link Proto    Local                 Remote
em0                     up    up
em0.0                   up    up   inet     172.17.0.2/16

{master:0}
antidote@cb3707b8151e> quit


cb3707b8151e (ttyd0)
```

Hit ^P^Q to exit the console, then shutdown the container with

```
docker kill vqfx
```

## Openconfig

Openconfig package can be added to the vqfx container images by placing in the current folder before building the containers.
So far the pacakge doesn't get automatically loaded at runtime. 

To load and activate OpenConfig in a running container, execute 
the following command:

```
master:0}
root@223bde03203e> request system software add /var/db/vmm/junos-openconfig-0.0.0.10-1-signed.tgz no-validate        
Verified junos-openconfig-0.0.0.10-1.tgz signed by PackageProductionEc_2018 method ECDSA256+SHA256
Adding junos-openconfig...
Mounted junos-openconfig package on /dev/md13...
Verified manifest signed by PackageProductionEc_2018 method ECDSA256+SHA256
Executing /packages/mnt/junos-openconfig-0.0.0.10-1/mount.post..
TLV generation: START
TLV generation: SUCCESS
Building schema and reloading /config/juniper.conf.gz ...
mgd: commit complete
Restarting mgd ...
Saving package file in /var/sw/pkg/junos-openconfig-0.0.0.10-1-signed.tgz ...
Saving state for rollback ...

WARNING: cli has been replaced by an updated version:
CLI release 18.1R1.9 built by builder on 2018-03-23 22:24:48 UTC
Restart cli using the new version ? [yes,no] (yes) 

Restarting cli ...
{master:0}
root@223bde03203e> 
```

The openconfg package is currently not compatible to 18.4R1. 

