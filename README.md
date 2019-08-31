![logo](https://i.imgur.com/tvDHssX.png)

# `rup` A reliable udp server/client package
*the name is based on the [rudp from plan 9](https://en.wikipedia.org/wiki/Reliable_User_Datagram_Protocol) but witout the p*

## But why?
Simply because there seems to be no library/protocol that works over udp, works like as easy as an http request, can do UDP hole punching, is **not** bloated and not focused on web.  
Thinks i've considered before making this:    
- **WebRTC** - Could work but there focus is realtime video/audio streams on the web
- **Stun/Turn** - Could work but i don't like the stun nor turn protocol
- **HTTP/3 (quic)** - Probebly one of the best alternatives but there main focus is being a web protocol what is one of the things i don't want

And i like learning new things, this seemed like a good chelange :),  

## Feathers
- Fast
- Reliable
- Support for UDP hole punching
