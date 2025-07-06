> [!WARNING]  
> This idea/fork are currently put on hold and can be removed at any moment.
> `gateway/` code seems to work fine but we cannot exactly figure out why Discord Gateway always drops our attempts at resuming session.
> Shards reliably read/send payloads, control heartbeat and tries to resume session whenever possible **but still - they end up having to open new session every time.**
> This bug may be caused by our code or it can very well be incorrect information on Discord Developers website.

### Alternative lib: Qord

I've forked Tempest's master branch with idea of having alternative client that uses discord gateway instead http for people who wants very low latency and don't care about having to keep connections alive (they host bot on their own VPS instead cloud, etc.).

### Ideas over master branch
- Use gateway to halve latency,
- Instantly update bot's avatar & banner images thanks to constant heartbeat

![Example working code](https://media.discordapp.net/attachments/1387847872879398952/1391454066940313661/screenshot.png?ex=686bf415&is=686aa295&hm=88ab58911fa2ec15471a74190e47081ee76d44d76015963be73dd9d6655a64e9&=&format=webp&quality=lossless&width=1474&height=829)
