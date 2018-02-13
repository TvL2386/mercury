package zdns

import (
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"
)

// Get returns a dns record from cache
func (c *Cache) Get(domainName string, queryType string, hostName string, client net.IP, honorTTL bool) ([]Record, error) {
	c.RLock()
	defer c.RUnlock()
	searchDomain := strings.ToLower(domainName)
	searchHostname := strings.ToLower(hostName)
	if _, ok := c.Domain[searchDomain]; ok {
		if _, ok := c.Domain[searchDomain].QueryType[queryType]; ok {
			if _, ok := c.Domain[searchDomain].QueryType[queryType].HostRecord[searchHostname]; ok {
				records := []Record{}
				balanceMode := ""
				for _, record := range c.Domain[searchDomain].QueryType[queryType].HostRecord[searchHostname] {
					if record.Type == "SOA" {
						reg, _ := regexp.Compile("###([A-Z_a-z]+)###")
						fn := func(m string) string {
							p := reg.FindStringSubmatch(m)
							switch p[1] {
							case "SERIAL":
								return fmt.Sprintf("%d", time.Now().Unix()-(time.Now().Unix()%10))
							}
							return m
						}
						record.Target = reg.ReplaceAllStringFunc(record.Target, fn)
					}
					if record.Online {
						if record.ttlExpire.After(time.Now()) || honorTTL == false {
							// apply same 0x20 encoding as requested fqdn
							record.Domain = domainName
							record.Name = hostName

							if honorTTL {
								record.TTL = int(-time.Since(record.ttlExpire).Seconds())
							}

							records = append(records, record)
							balanceMode = record.BalanceMode
						}
					}
				}
				if len(records) == 0 {
					return records, ErrNotFound
				}
				var err error
				if balanceMode != "" {
					records, err = MultiSort(records, client, balanceMode)
				}
				return records, err
			}
		}
	}
	return []Record{}, ErrNotFound
}

// GetAll returns a dns record from cache
func (c *Cache) GetAll(domainName string, client net.IP, honorTTL bool) ([]Record, error) {
	c.RLock()
	defer c.RUnlock()
	searchDomain := strings.ToLower(domainName)
	records := []Record{}
	if _, ok := c.Domain[searchDomain]; ok {
		for _, qd := range c.Domain[searchDomain].QueryType {
			for _, hd := range qd.HostRecord {
				for _, record := range hd {
					if record.Type == "SOA" {
						reg, _ := regexp.Compile("###([A-Z_a-z]+)###")
						fn := func(m string) string {
							p := reg.FindStringSubmatch(m)
							switch p[1] {
							case "SERIAL":
								return fmt.Sprintf("%d", time.Now().Unix()-(time.Now().Unix()%10))
							}
							return m
						}
						record.Target = reg.ReplaceAllStringFunc(record.Target, fn)
					}
					if record.Online {
						records = append(records, record)
					}
				}
			}
		}
	}
	var err error
	if len(records) == 0 {
		err = ErrNotFound
	}
	return records, err
}

// IsServedDomain returns true or false, if we serve requests in this domain
func (c *Cache) IsServedDomain(domain string) bool {
	c.RLock()
	defer c.RUnlock()
	if _, ok := c.Domain[strings.ToLower(domain)]; ok {
		return true
	}
	return false
}
