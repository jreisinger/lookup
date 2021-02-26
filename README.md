Lookup FQDN at many [public DNS servers](https://public-dns.info/nameservers.txt).

```
$ go install

$ lookup -n 10 example.com 2> /dev/null # ignore failed nameservers
lookup at 1.0.0.1         OK
lookup at 8.8.4.4         OK
lookup at 1.1.1.1         OK
lookup at 8.8.8.8         OK
lookup at 10.235.119.38   OK
lookup at 10.235.119.37   OK
lookup at 119.160.80.164  OK
7 ok responses out of 7 (100.00%)

$ echo $? # exit non-zero if there's less than 90% ok responses
0
```