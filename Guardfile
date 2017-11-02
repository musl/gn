# vim: set ft=ruby ts=2 sw=2:

options = {
  :name => 'make',
  :command => 'make',
  :env => {},
}

interactor :off

guard( :process, options ) do
  watch( /.*\.(go)$/ )
end
