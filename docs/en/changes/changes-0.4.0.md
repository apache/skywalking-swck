## 0.4.0

#### Features
- Add Java agent injector.
- Add JavaAgent and Storage CRDs of the operator.

### Vulnerabilities

- CVE-2021-3121: An issue was discovered in GoGo Protobuf before 1.3.2. plugin/unmarshal/unmarshal.go lacks certain index validation
- CVE-2020-29652: A nil pointer dereference in the golang.org/x/crypto/ssh component through v0.0.0-20201203163018-be400aefbc4c for Go allows remote attackers to cause a denial of service against SSH servers.

#### Chores
- Bump up GO to 1.17.
- Bump up k8s api to 0.20.11.
- Polish documents.
- Bump up SkyWalking OAP to 8.8.1.
