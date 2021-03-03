/*
 * Autopolicy PoC
 * Copyright (C) 2020-2020 IITiS PAN Gliwice <https://www.iitis.pl/>
 * Author: Piotr Frohlich <pfrohlich@iitis.pl>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */
package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
)

var opts struct {
	ResolverIP string `short:"r" long:"resolver" description:"IP of the DNS resolver to use for lookups"`
	Protocol   string `short:"P" long:"protocol" choice:"tcp" choice:"udp" default:"udp" description:"Protocol to use for lookups"`
	Port       uint16 `short:"p" long:"port" default:"53" description:"Port to bother the specified DNS resolver on"`
	Domain     bool   `short:"d" long:"domain" description:"Output only domains"`
}

func resolve(address string, reverse bool) ([]string, error) {
	var r *net.Resolver
	if opts.ResolverIP != "" {
		r = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, adr string) (net.Conn, error) {
				d := net.Dialer{}
				return d.DialContext(ctx, "udp", fmt.Sprintf("%s:%d", opts.ResolverIP, opts.Port))
			},
		}
	}
	if !reverse {
		u, err := url.Parse(address)
		if err != nil {
			return nil, err
		}
		if u.Host == "" {
			if strings.Count(address, "/") > 0 {
				parts := strings.Split(address, "/")
				fmt.Println("parts ", parts)
				for _, s := range parts {
					if strings.Contains(s, ".") {
						u.Host = s
						break
					}
				}
			} else {
				u.Host = address
			}
		}
		addr, err := r.LookupIP(context.Background(), "ip4", u.Host)
		if err != nil {
			return nil, err
		}
		ret := make([]string, 0)
		for _, a := range addr {
			ret = append(ret, a.String())
		}
		return ret, nil
	} else {
		addr, err := r.LookupAddr(context.Background(), address)
		if err != nil {
			return nil, err
		}
		return addr, nil
	}
}
func configureDNS(resolver string, port uint16) {
	opts.ResolverIP = resolver
	opts.Port = port
}

func resolveProfile(profile Profile, reverse bool) (Profile, error) {
	allows := make([]string, 0)
	fromDevice, ok := profile["from_device"].(map[string]interface{})
	if !ok {
		return nil, errors.New("Unable to get from device")
	}
	allow, ok := fromDevice["allow"].([]interface{})
	if !ok {
		allows, ok = fromDevice["allow"].([]string)
		if !ok {
			return nil, errors.New("Unable to parse allow (both string and interface)")
		}
	} else {
		for _, i := range allow {
			allows = append(allows, fmt.Sprint(i))
		}
	}
	dbg(2, "Resolve Profile", "Parsed profile to "+fmt.Sprint(allows))
	var flowsAll []string
	for _, flow := range allows {
		destination := strings.Split(flow, " ")[1]
		var addresses []string
		if isIP(destination) {
			if reverse {
				addresses, _ = resolve(destination, true)
			}
		} else if !reverse {
			addresses, _ = resolve(destination, false)
		}
		if addresses == nil {
			flowsAll = append(flowsAll, flow)
		} else {
			for _, addr := range addresses {
				flowsAll = append(flowsAll, strings.Replace(flow, destination, addr, -1))
			}
			dbg(3, "Resolver", "Added "+fmt.Sprint(len(addresses))+" reverse resolved addresses.")
		}
	}
	dbg(2, "Resolver", "Resolved profile"+fmt.Sprint(flowsAll))
	fd := make(map[string]interface{})
	al := make(map[string]interface{})
	al["allow"] = flowsAll
	fd["from_device"] = al
	return fd, nil

}
func isIP(destination string) bool {
	re := regexp.MustCompile(`(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}`)
	return re.MatchString(destination)
}
