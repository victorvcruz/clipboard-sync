#!/bin/bash

# Script para mostrar os IPs disponíveis do dispositivo

echo "📍 Your current IP addresses:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Verifica se o comando ip está disponível
if command -v ip &> /dev/null; then
    # Usa o comando ip (mais moderno)
    ip addr show | grep -E "inet [0-9]" | grep -v "127.0.0.1" | while read -r line; do
        interface=$(echo "$line" | awk '{print $NF}')
        ip_addr=$(echo "$line" | awk '{print $2}' | cut -d'/' -f1)
        
        # Detecta o tipo de rede
        if [[ $ip_addr == 192.168.* ]] || [[ $ip_addr == 10.* ]] || [[ $ip_addr == 172.1[6-9].* ]] || [[ $ip_addr == 172.2[0-9].* ]] || [[ $ip_addr == 172.3[0-1].* ]]; then
            network_type="(Local Network)"
        else
            network_type="(Public)"
        fi
        
        echo "  Interface $interface: $ip_addr $network_type"
    done
elif command -v ifconfig &> /dev/null; then
    # Fallback para ifconfig (mais antigo)
    ifconfig | grep -E "inet [0-9]" | grep -v "127.0.0.1" | while read -r line; do
        ip_addr=$(echo "$line" | awk '{print $2}')
        
        # Detecta o tipo de rede
        if [[ $ip_addr == 192.168.* ]] || [[ $ip_addr == 10.* ]] || [[ $ip_addr == 172.1[6-9].* ]] || [[ $ip_addr == 172.2[0-9].* ]] || [[ $ip_addr == 172.3[0-1].* ]]; then
            network_type="(Local Network)"
        else
            network_type="(Public)"
        fi
        
        echo "  $ip_addr $network_type"
    done
else
    echo "❌ Nem 'ip' nem 'ifconfig' estão disponíveis"
    echo "   Tente instalar: sudo apt-get install net-tools iproute2"
fi

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "💡 Para sincronização entre dispositivos na mesma rede,"
echo "   use o IP da rede local (192.168.x.x, 10.x.x.x, ou 172.x.x.x)"
echo ""
echo "Exemplo de uso:"
echo "  clipboard-sync -target 192.168.1.100"