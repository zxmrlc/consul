import TabsBase from '@hashicorp/react-tabs'
import s from '@hashicorp/nextjs-scripts/lib/providers/docs/style.module.css'
import EnterpriseAlertBase from '@hashicorp/react-enterprise-alert'

// Tabs is a general-purpose component that we format for ease of use within mdx
// It is also wrapped in a span with a css module class for styling overrides
function Tabs({ children }) {
  if (!Array.isArray(children))
    throw new Error('Multiple <Tab> elements required')

  return (
    <span className={s.tabsRoot}>
      <TabsBase
        items={children.map((Block) => ({
          heading: Block.props.heading,
          // eslint-disable-next-line react/display-name
          tabChildren: () => Block,
        }))}
      />
    </span>
  )
}

// This is a simple helper that makes it more clear for docs authors
function Tab({ children }) {
  return <>{children}</>
}

function EnterpriseAlert(props) {
  return <EnterpriseAlertBase product={'consul'} {...props} />
}

// toYAMLKey converts an HCL key to a kube yaml key.
// e.g. Protocol => protocol, MeshGateway => meshGateway, ACLToken => aclToken.
// indexFirstLowercaseChar=1
// indexFirstLowercaseChar=1
// indexFirstLowercaseChar=4
function toYAMLKey(hclKey) {
  let indexFirstLowercaseChar = 0
  for (let i = 0; i < hclKey.length; i++) {
    if (hclKey[i].toLowerCase() === hclKey[i]) {
      indexFirstLowercaseChar = i
      break
    }
  }

  if (indexFirstLowercaseChar > 1) {
    indexFirstLowercaseChar--
  }

  let yamlKey = ''
  for (let i = 0; i < indexFirstLowercaseChar; i++) {
    yamlKey += hclKey[i].toLowerCase()
  }
  yamlKey += hclKey.split('').slice(indexFirstLowercaseChar).join('')

  return yamlKey
}

function renderNode(node, isHCLTab) {
  let key = node.key
  if (!key) {
    return null
  }
  if (!isHCLTab) {
    key = toYAMLKey(key)
  }
  if (isHCLTab && node.hcl === false) {
    return null
  }
  if (!isHCLTab && node.yaml === false) {
    return null
  }
  const keyLower = key.toLowerCase()

  let description = ''
  if (!isHCLTab && node.kubeDescription) {
    description = node.kubeDescription
  } else if (node.description) {
    description = node.description
  }

  let htmlDescription = ''
  if (description !== '') {
    htmlDescription = '- ' + description
    while (htmlDescription.indexOf('`') > 0) {
      htmlDescription = htmlDescription.replace('`', '<code>')
      if (htmlDescription.indexOf('`') <= 0) {
        throw new Error(
          "description: '" +
            description +
            "' does not have matching '`' characters"
        )
      }
      htmlDescription = htmlDescription.replace('`', '</code>')
    }
  }

  let type = ''
  if (node.type) {
    type = <code>{'(' + node.type + ')'}</code>
  }

  let enterpriseAlert = ''
  if (node.enterprise) {
    enterpriseAlert = <EnterpriseAlert inline />
  }

  return (
    <li key={keyLower} className="g-type-long-body">
      <a id={keyLower} className="__target-lic" aria-hidden="" />
      <p>
        <a
          href={'#' + keyLower}
          aria-label={keyLower + ' permalink'}
          className="__permalink-lic"
        >
          <code>{key}</code>
        </a>{' '}
        {type}
        {enterpriseAlert}
        <span dangerouslySetInnerHTML={{ __html: htmlDescription }} />
      </p>
      {renderNodes(node.children, isHCLTab)}
    </li>
  )
}

function renderNodes(nodes, isHCLTab) {
  if (!nodes) {
    return null
  }

  return (
    <ul>
      {nodes.map((node) => {
        return renderNode(node, isHCLTab)
      })}
    </ul>
  )
}

function isTopLevelKubeKey(key) {
  return (
    key.toLowerCase() === 'metadata' ||
    key.toLowerCase() === 'kind' ||
    key.toLowerCase() === 'apiversion'
  )
}

export default function ConfigEntrySpec({ spec }) {
  // Kube needs to have its non-top-level nodes nested under a "spec" key.
  const topLevelNodes = spec.filter((node) => {
    return isTopLevelKubeKey(node.key)
  })
  const nodesUnderSpec = spec.filter((node) => {
    return !isTopLevelKubeKey(node.key)
  })
  const kubeNodes = topLevelNodes.concat([
    { key: 'spec', children: nodesUnderSpec },
  ])

  return (
    <Tabs>
      <Tab heading="HCL">{renderNodes(spec, true)}</Tab>
      <Tab heading="Kubernetes YAML">{renderNodes(kubeNodes, false)}</Tab>
    </Tabs>
  )
}
