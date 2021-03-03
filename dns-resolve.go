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
	"fmt"
	flags "github.com/jessevdk/go-flags"
	"net"
	"net/url"
	"os"
	"strings"
)

var opts struct {
	ResolverIP string `short:"r" long:"resolver" description:"IP of the DNS resolver to use for lookups"`
	Protocol   string `short:"P" long:"protocol" choice:"tcp" choice:"udp" default:"udp" description:"Protocol to use for lookups"`
	Port       uint16 `short:"p" long:"port" default:"53" description:"Port to bother the specified DNS resolver on"`
	Domain     bool   `short:"d" long:"domain" description:"Output only domains"`
}

func resolve(address string, params [] string, reverse bool) ([] string,error){
	_, err := flags.ParseArgs(&opts, params)
	if err != nil{
		os.Exit(1)
	}
	var r *net.Resolver
	if opts.ResolverIP != "" {
		r = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{}
				return d.DialContext(ctx, opts.Protocol, fmt.Sprintf("%s:%d", opts.ResolverIP, opts.Port))
			},
		}
	}
	if !reverse{
		u, err := url.Parse(address)
		if err!=nil{
			return nil, err
		}
		if u.Host==""{
			if strings.Count(address,"/")>0{
				parts:=strings.Split(address,"/")
				fmt.Println("parts ",parts)
				for _,s := range parts{
					if strings.Contains(s,"."){
						u.Host = s
						break
					}
				}
			} else {
				u.Host = address
			}
		}
		addr,err:=r.LookupIP(context.Background(),"ip4",u.Host)
		if err!=nil{
			return nil, err
		}
		ret := make([]string,1)
		for _,a := range addr{
			ret = append(ret,a.String())
		}
		return ret,nil
	}else {
		addr, err := r.LookupAddr(context.Background(), address)
		if err != nil {
			return nil,err
		}
		return addr,nil
	}
}