
We hebben de situatie zoals deze zich voordeed na het laatste onderhoud
onderzocht. We hebben zoveel mogelijk scenario's nagebootst in situaties met
secundaire IP adressen en floating IP adressen, zowel op IPv4 als IPv6.

We hebben het probleem zoals dat bij jullie voordeed niet kunnen reproduceren.
Wel zien we dat de switch waarop we het onderhoud deden er de nodige tijd over
deed om alle MAC adressen en de daaraan geassocieerde IP adressen (ARP/NDP) weer
te leren vanuit de clients. In sommige gevallen duurde dit wel tot een half uur
na het oorspronkelijke verlies van connectiviteit.

Een interessante bevinden waar we tegenaanliepen terwijl we jullie configuratie
onderzochten, is dat jullie VPS'en een interface hebben in VLAN 2. Dat is het
VLAN dat wij gebruiken voor global fail-over VIPs; zowel IPv4 als IPv6. De IP
adressen die aan deze VPS'en gekoppeld zijn, zijn echter allemaal geen global
fail-over VIP's maar losse IP adressen binnen het pod waar de machines in
zitten. 

Wij vermoeden dat hier tijdens de set-up iets niet goed is gecommuniceerd, maar
dat kunnen we helaas niet valideren zonder een volledig overzicht van de
configuratie. Het zou wel goed kunnen dat jullie machines - in tegenstelling tot
die van ons en andere klanten - hierdoor langer offline zijn gebleven.

We hebben hiernaast nog een issue geconstateerd in de configuratie van
keepalived in de IPv6 setup. Alle IPv6 adressen die in de keepalived
configuratie van lb3 en lb4 geconfigureerd zijn, behoren exclusief toe aan de
netwerk interface van VPS lb3, en zijn derhalve niet geschikt als VIP.

Het is niet mogelijk om de IPv6-adressen van lb3 op een andere VPS zoals lb4
actief te maken; de NDP entry voor het IPv6 adres in combinatie met het MAC
adres van lb4 zal dan ook niet worden geregistreerd door de switches, waardoor
verkeer niet mogelijk is.

Voor ieder MAC adres en de daarbijbehorende IP-adressen van een VPS worden
firewallregels toegepast, die alleen verkeer voor die specifieke combinatie
toestaan. Dit in tegenstelling tot VIP-adressen die door meerdere VPSen gebruikt
mogen worden. Deze constructie is voornamelijk bedoeld om ervoor te zorgen dat
VPSen met een misconfiguratie geen andere VPSen kunnen verstoren. Dit kunnen we
indien gewenst vrij eenvoudig oplossen door deze IPv6 adressen los te koppelen
van lb3.

Naast deze zaken zien we ook nog een probleem in onze switch, waarvoor jullie
ook ticket #125435 hebben aangemaakt. Hierbij lijkt de route naar de switch om
welke reden dan ook niet bekend te worden bij sommige VPS'en. Wanneer de VPS de
gateway het global public IPv4 adres van de switch pingt, wordt de route wel
bekend en vanaf dat moment zien wij geen problemen meer.

We hebben support gevraagd van Juniper om mee te kijken in dit issue, maar tot
op heden hebben we de oorzaak nog niet weten te vinden. Wel is het zo dat we
deze switch ook niet lang meer zullen gebruiken, omdat we in de komende maanden
zullen migreren naar een nieuw netwerk. Als we meer leren over de oorzaak van
dit probleem, zullen we dit met jullie delen.

