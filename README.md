# longview-go
This is going to be a GO version of Linode Longview agent (client): 

https://github.com/linode/longview

I like the software, but don't like that it's written in Perl and installed
a whole bunch of Perl libraries (modules) as dependencies.

A Go version will just be a single executable, as usual.

Things that are working:

- CPU (user / system times, load average)
- SysInfo (processor count, model, kernel version)
- Memory (free / used for real and swap)

The goal is to implement everything which is in the Perl version, so these will
come next and fairly soon I hope:

- Network
- Disks
- Processes

And then

- Package updates
- NGINX

- Apache (???) I don't use it and don't have it installed, but will take a pull request
