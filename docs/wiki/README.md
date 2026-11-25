# Wiki de LibreQuenda

LibreQuenda é a reserva de instalacións deportivas municipais dos concellos galegos: software libre, sen depender dun provedor e pensado para quen reserva e para quen xestiona.

Esta wiki ensina o que se pode facer, pantalla a pantalla, segundo quen usa a aplicación:

| | Quen | Onde | Que fai |
|---|---|---|---|
| 🏃 | [**Veciñanza**](vecinanza.md) | App Android e web | Atopa onde xogar, reserva, entra na instalación sen chaves, valora, avisa de problemas e responde ás enquisas do concello |
| 🏛️ | [**Concello**](concello.md) | Web, en `/<concello>/admin` | Dá de alta as instalacións, os horarios e as normas, verifica a veciñanza, organiza eventos, publica comunicados e enquisas e mide o uso |
| 🛠️ | [**Superadministración**](superadmin.md) | Web, en `/admin` | Crea os concellos, mantén o catálogo de etiquetas e a imaxe da plataforma, e revisa o rexistro |

<p align="center">
  <img src="../screenshots/android/vecino/facilities/home.webp" width="230" alt="Inicio da app">
  <img src="../screenshots/android/vecino/facilities/facilities-map.webp" width="230" alt="Mapa de instalacións">
  <img src="../screenshots/android/vecino/flash/07-booked-and-in.webp" width="230" alt="Reserva flash feita e entrada rexistrada">
</p>

## Antes de empezar

- As capturas son dunha instalación local de proba co concello da Coruña e datos inventados: as persoas, as reservas e as valoracións non son reais.
- A app Android é só para a veciñanza; o concello e a superadministración traballan na web. Cando unha acción do concello chega á veciñanza (un aviso, un correo), a captura da app vai xunto á do concello.
- As capturas están en [`docs/screenshots`](../screenshots), en WebP, por rol e por función.
- A lista completa de funcións e o seu estado está en [`features.md`](../../features.md).

## Catálogos de software público

O repositorio leva un [`publiccode.yml`](../../publiccode.yml), o ficheiro [estándar](https://yml.publiccode.tools) que len os catálogos de software para as administracións públicas (como o de Developers Italia). Describe LibreQuenda en galego, castelán e inglés: que fai, para quen é, as súas funcións e capturas, a licenza (AGPL-3.0), quen o mantén, os idiomas e as dependencias. Así un concello que busca software libre para reservar instalacións deportivas pode atopalo e compararlo sen ter que ler o código.

Se cambian as funcións, as capturas, a licenza ou o contacto, hai que actualizalo tamén. Comprobar que segue sendo válido:

```sh
go run github.com/italia/publiccode-parser-go/v5/publiccode-parser@latest publiccode.yml
```
