# Comparaison visuelle locale avec main

Base : `7a28d795930b1f34201f04e5a071247c7949811a`. Même environnement, Chrome 156, polices et fixtures synthétiques. Les quatre tests échouent par rapport aux références CI dans les deux versions. Les PNG refonte/main sont identiques octet pour octet. Aucun seuil ou fichier de référence modifié.

| Vue | Dimensions (px) | Pixels différents | SHA-256 identique des deux PNG |
|---|---|---|---|
| dashboard | 1212 × 3221 | 0 / 3903852 | `4a402ddec224a2f9678afdc606b094ec285bbbb96c7aac657cb5ca8b7344f0f6` |
| transactions | 1212 × 6650 | 0 / 8059800 | `a461c83b10de7a19df7dccf36f44a5c0e6a2cd7f0ac0c0855fbd150fed989f3d` |
| budgets | 1212 × 1158 | 0 / 1403496 | `cdc2bb12cf6a377e7e1073afacae5acaebfe6f5275aba5d012b4a4f1e088e47d` |
| simulation | 1212 × 1069 | 0 / 1295628 | `f2438968068aeeede50a20b26a1bf679f1434dc780bf173029fdbb58bca523c5` |
