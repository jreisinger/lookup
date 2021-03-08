Look up FQDN at many [public DNS servers][1] and report statistics.

```
$ go install
```

```
$ lookup -n 10 -t mx example.com
querying 8.8.8.8         1 RR (NOERROR)
querying 8.8.4.4         1 RR (NOERROR)
querying 1.0.0.1         1 RR (NOERROR)
querying 1.1.1.2         1 RR (NOERROR)
querying 1.1.1.1         1 RR (NOERROR)
querying 151.80.222.79   1 RR (NOERROR)
querying 200.11.52.202   1 RR (NOERROR)
querying 82.146.26.2     read udp 192.168.100.92:63755->82.146.26.2:53: i/o timeout
querying 94.236.218.254  read udp 192.168.100.92:58031->94.236.218.254:53: i/o timeout
querying 199.255.137.34  read udp 192.168.100.92:59764->199.255.137.34:53: i/o timeout
----------------------------------------
Failed nameservers       30% (3/10)
Failed responses          0% (0/7)
Empty responses           0% (0/7)
```

[1]: https://public-dns.info/nameservers.txt