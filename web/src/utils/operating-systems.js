// 操作系统数据
export const operatingSystems = [
  // Linux 发行版
  { name: 'ubuntu', displayName: 'Ubuntu', category: 'Linux' },
  { name: 'debian', displayName: 'Debian', category: 'Linux' },
  { name: 'centos', displayName: 'CentOS', category: 'Linux' },
  { name: 'rhel', displayName: 'Red Hat Enterprise Linux', category: 'Linux' },
  { name: 'fedora', displayName: 'Fedora', category: 'Linux' },
  { name: 'opensuse', displayName: 'openSUSE', category: 'Linux' },
  { name: 'alpine', displayName: 'Alpine Linux', category: 'Linux' },
  { name: 'arch', displayName: 'Arch Linux', category: 'Linux' },
  { name: 'mint', displayName: 'Linux Mint', category: 'Linux' },
  { name: 'kali', displayName: 'Kali Linux', category: 'Linux' },
  { name: 'rocky', displayName: 'Rocky Linux', category: 'Linux' },
  { name: 'almalinux', displayName: 'AlmaLinux', category: 'Linux' },
  { name: 'oracle', displayName: 'Oracle Linux', category: 'Linux' },
  { name: 'amazonlinux', displayName: 'Amazon Linux', category: 'Linux' },
  { name: 'sles', displayName: 'SUSE Linux Enterprise Server', category: 'Linux' },
  { name: 'gentoo', displayName: 'Gentoo', category: 'Linux' },
  { name: 'void', displayName: 'Void Linux', category: 'Linux' },
  { name: 'nixos', displayName: 'NixOS', category: 'Linux' },
  
  // BSD 系统
  { name: 'freebsd', displayName: 'FreeBSD', category: 'BSD' },
  { name: 'openbsd', displayName: 'OpenBSD', category: 'BSD' },
  { name: 'netbsd', displayName: 'NetBSD', category: 'BSD' },
  { name: 'dragonflybsd', displayName: 'DragonFly BSD', category: 'BSD' },
  
  // Windows 系统
  { name: 'windows', displayName: 'Windows Server', category: 'Windows' },
  { name: 'windows2019', displayName: 'Windows Server 2019', category: 'Windows' },
  { name: 'windows2022', displayName: 'Windows Server 2022', category: 'Windows' },
  { name: 'windows10', displayName: 'Windows 10', category: 'Windows' },
  { name: 'windows11', displayName: 'Windows 11', category: 'Windows' },
  
  // 容器系统
  { name: 'busybox', displayName: 'BusyBox', category: 'Container' },
  { name: 'scratch', displayName: 'Scratch', category: 'Container' },
  { name: 'distroless', displayName: 'Distroless', category: 'Container' },
  
  // 其他系统
  { name: 'openwrt', displayName: 'OpenWrt', category: 'Embedded' },
  { name: 'other', displayName: '其他', category: 'Other' }
]

// 根据分类获取操作系统
export const getOperatingSystemsByCategory = () => {
  const grouped = {}
  operatingSystems.forEach(os => {
    if (!grouped[os.category]) {
      grouped[os.category] = []
    }
    grouped[os.category].push(os)
  })
  return grouped
}

// 根据名称获取操作系统信息
export const getOperatingSystemByName = (name) => {
  return operatingSystems.find(os => os.name === name)
}

// 获取所有操作系统名称列表
export const getAllOperatingSystemNames = () => {
  return operatingSystems.map(os => os.name)
}

// 获取显示名称
export const getDisplayName = (name) => {
  const os = getOperatingSystemByName(name)
  return os ? os.displayName : name
}

// 常用操作系统版本映射
export const commonVersions = {
  ubuntu: ['20.04', '22.04', '24.04', '18.04'],
  debian: ['11', '12', '10', '9'],
  centos: ['7', '8', '9'],
  rhel: ['8', '9', '7'],
  fedora: ['38', '39', '40', '37'],
  opensuse: ['15.5', '15.4', 'Tumbleweed'],
  alpine: ['3.18', '3.19', '3.17', 'edge'],
  arch: ['latest'],
  mint: ['21', '20.3', '19.3'],
  kali: ['2023.4', '2023.3', 'rolling'],
  rocky: ['8', '9'],
  almalinux: ['8', '9'],
  oracle: ['8', '9', '7'],
  amazonlinux: ['2', '2023'],
  sles: ['15', '12'],
  freebsd: ['13.2', '12.4', '14.0'],
  openbsd: ['7.4', '7.3'],
  netbsd: ['9.3', '10.0'],
  windows2019: ['1809', '1903', '1909'],
  windows2022: ['21H2'],
  windows10: ['21H2', '22H2'],
  windows11: ['21H2', '22H2', '23H2'],
  openwrt: ['22.03', '23.05']
}

// 根据操作系统获取常用版本
export const getCommonVersions = (osName) => {
  return commonVersions[osName] || []
}
