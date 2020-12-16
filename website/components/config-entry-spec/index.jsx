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

function renderNode(node, keyKey) {
  const key = node[keyKey]
  if (!key) {
    return null
  }
  const keyLower = key.toLowerCase()

  let description = ''
  if (keyKey === 'kubeKey' && node.kubeDescription) {
    description = ' - ' + node.kubeDescription
  } else if (node.description) {
    description = ' - ' + node.description
  }

  let type = ''
  if (node.type) {
    type = <code>({node.type})</code>
  }

  let enterpriseAlert = ''
  if (node.enterprise) {
    enterpriseAlert = <EnterpriseAlert inline />
  }

  return (
    <li key={keyLower} className="g-type-long-body">
      <a id={'#' + keyLower} className="__target-lic" aria-hidden="" />
      <p>
        <a href="#kind" aria-label="kind permalink" className="__permalink-lic">
          <code>{key}</code>
        </a>{' '}
        {type}
        {enterpriseAlert}
        <>{description}</>
      </p>
      {renderNodes(node.children, keyKey)}
    </li>
  )
}

function renderNodes(nodes, keyKey) {
  if (!nodes) {
    return null
  }

  const renderedNodes = nodes.map((node) => {
    return renderNode(node, keyKey)
  })
  return <ul>{renderedNodes}</ul>
}

export default function ConfigEntrySpec({ spec }) {
  // Kube needs to have its non-top-level nodes nested under a "spec" key.
  const topLevelNodes = spec.filter((node) => {
    return (
      node.kubeKey === 'metadata' ||
      node.kubeKey === 'kind' ||
      node.kubeKey === 'apiVersion'
    )
  })
  const nodesUnderSpec = spec.filter((node) => {
    return !(
      node.kubeKey === 'metadata' ||
      node.kubeKey === 'kind' ||
      node.kubeKey === 'apiVersion'
    )
  })
  const kubeNodes = topLevelNodes.concat([
    { kubeKey: 'spec', children: nodesUnderSpec },
  ])

  return (
    <Tabs>
      <Tab heading="HCL">{renderNodes(spec, 'key', 'description')}</Tab>
      <Tab heading="Kubernetes YAML">{renderNodes(kubeNodes, 'kubeKey')}</Tab>
    </Tabs>
  )
}
