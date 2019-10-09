# longview-go
This is a GO version of Linode Longview agent (client): 

https://github.com/linode/longview

I like the software, but don't like that it's written in Perl and installs
a whole bunch of Perl libraries (modules) as dependencies.

This Go version is just a single executable, no dependencies.

Things that are working now:

- CPU (user / system times, load average)
- SysInfo (processor count, model, kernel version)
- Memory (free / used for real and swap)
- Network, t/rx and MAC
- Network, listen sockets and connections
- Disks
- Processes
- NGINX
- Apache
- MySQL

At this time these are still missing compared to Linode's official version:

- Package updates

- Support for dm/md disk devices (LVM?)

# License

Since it's based on the original Linode Longview Perl code - the license for
this (derived work) is GPL 

# Source code

- Main: https://github.com/kmansoft/longview-go

- Mirror: https://gitlab.com/kmansoft/longview-go
