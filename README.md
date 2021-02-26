Look up FQDN at many [public DNS servers](https://public-dns.info/nameservers.txt) and report success rate.

```
$ go install

$ lookup -n 10 -t mx example.com 2> /dev/null # ignore failed nameservers
response from 1.0.0.1         contained 1 RR
response from 8.8.8.8         contained 1 RR
response from 1.1.1.2         contained 1 RR
response from 8.8.4.4         contained 1 RR
response from 1.1.1.1         contained 1 RR
response from 119.160.80.164  contained 1 RR
response from 151.80.222.79   contained 1 RR
100% (7/7) of responses contained resource records

$ echo $? # exit non-zero if less than 90% of responses contained RRs
0
```