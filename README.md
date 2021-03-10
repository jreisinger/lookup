Look up FQDN at many [public DNS servers][1] concurrently and report statistics.

```
$ go install
```

```
$ lookup -n 10 -t mx example.com
lookup at 8.8.4.4         1 RR
lookup at 8.8.8.8         1 RR
lookup at 1.1.1.2         1 RR
lookup at 1.1.1.1         1 RR
lookup at 1.0.0.1         1 RR
lookup at 119.160.80.164  1 RR
lookup at 151.80.222.79   read udp 192.168.100.92:54121->151.80.222.79:53: i/o timeout
lookup at 94.236.218.254  read udp 192.168.100.92:61781->94.236.218.254:53: i/o timeout
lookup at 199.255.137.34  read udp 192.168.100.92:49548->199.255.137.34:53: i/o timeout
lookup at 82.146.26.2     read udp 192.168.100.92:54256->82.146.26.2:53: i/o timeout
----------------------------------------
Failed nameservers         40% (4/10)
Failed responses            0% (0/6)
Empty responses             0% (0/6)
```

[1]: https://public-dns.info/nameservers.txt
