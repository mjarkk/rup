# `rup` A reliable udp server/client package
*the name is based on the [rudp from plan 9](https://en.wikipedia.org/wiki/Reliable_User_Datagram_Protocol) but witout the p*

## But why?
Simply because there seems to be no library/protocol that works over udp, works like a http request, can do UDP hole punching, is not bloated and not focused on web.  
Alternatives i've considered:    
- **WebRTC** - Could work but there focus is realtime video/audio streams on the web
- **Stun/Turn** - Could work but i don't like the stun nor turn protocol
- **HTTP/3 (quic)** - Probebly one of the best alternatives but there main focus is being a web protocol what is one of the things i don't want

And i like learning new things and this seemed like a good chelange :),  

## Feathers
- Fast
- Reliable
- Support for UDP hole punching

## How does the protocol work
*NOTE: This can be changed every  * 

### Messages send between

#### Data Messages
Data messages are the most important messages of this library because they contain the data send between the host and client
```
[meta] = The package meta data (max 1 char) + value (not needed) with "," as seperator and MUST end with a null byte ("\0"): "s,l:123\0" or just "\0" if you don't need any meta data
[data] = The actual data, can have a max length of (2000 - length of meta - 20) where 2000 is the udp recife size and 20 is the default lenght of a sha1 hash
[followup] = The hash of the next message it's data, must be in sha1

> The most basic message (this asumes that data is less than the max data length)
[meta] [data]

> If the data is more than the the max data part length, a message will look like this
> This message must be 2000 bytes, So the data also must be at it's max size 
[meta] [data] [hash of next message]
        
```
1. Meta data
2. Data
3. Hash of next message

#### Missing Message
Because UDP is not reliable by design there will be problems with packages that will never there destination  
That's why this library has a message for asking for missing data
