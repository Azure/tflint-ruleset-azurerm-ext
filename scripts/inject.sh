InjectPath="terraform-provider-azurerm/provider"
if [ ! -d ${InjectPath} ]; then
  cp -r provider terraform-provider-azurerm
  sh scripts/tmp2go.sh ${InjectPath}
fi
