# longview-go
This is going to be a GO version of Linode Longview agent (client): 

https://github.com/linode/longview

I like the software, but don't like that it's written in Perl and installed
a whole bunch of Perl libraries (modules) as dependencies.

A Go version will just be a single executable, as usual.

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

My goal is feature parity with the Perl version, at this time these are
still missing:

- Package updates

- Support for dm/md disk devices (LVM?)

# License

Since it's based on the original Linode Longview Perl code - the license for
this (derived work) is GPL 

# Source code

- Main: https://github.com/kmansoft/longview-go

- Mirror: https://gitlab.com/kmansoft/longview-go
