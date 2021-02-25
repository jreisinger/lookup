Lookup FQDN at many [public DNS servers](https://public-dns.info/nameservers.txt).

```
$ go install

$ lookup golang.org
lookup: lookup at 180.76.76.76    OK
lookup: lookup at 201.144.183.147 OK
lookup: lookup at 212.43.98.12    OK
<...SNIP...>
lookup: failed response from 0 out of 180 servers (0.00%)

$ echo $?
0
```