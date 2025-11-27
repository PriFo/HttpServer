import { useState, useCallback } from 'react'

export interface KPVEDNode {
  code: string
  name: string
  level: number
  children?: KPVEDNode[]
  has_children?: boolean
  item_count?: number
  parent_code?: string
}

export interface SearchResult {
  code: string
  name: string
  level: number
  parent_code?: string
}

export interface KPVEDStats {
  total: number
  levels: number
}

export function useKPVEDTree() {
  const [hierarchy, setHierarchy] = useState<KPVEDNode[]>([])
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set())
  const [loadingNodes, setLoadingNodes] = useState<Set<string>>(new Set())
  const [selectedNode, setSelectedNode] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [searchResults, setSearchResults] = useState<SearchResult[]>([])
  const [showSearchResults, setShowSearchResults] = useState(false)
  const [stats, setStats] = useState<KPVEDStats | null>(null)
  const [filterLevel, setFilterLevel] = useState<number | null>(null)
  const [nodePath, setNodePath] = useState<string[]>([])
  const [copiedCode, setCopiedCode] = useState<string | null>(null)

  const loadRootHierarchy = useCallback(async () => {
    try {
      const response = await fetch(`/api/kpved/hierarchy`)
      if (!response.ok) throw new Error('Failed to load hierarchy')

      const data = await response.json()
      setHierarchy(data.nodes || [])
      return data
    } catch (error) {
      console.error('Error loading root hierarchy:', error)
      throw error
    }
  }, [])

  const loadChildNodes = useCallback(async (parentCode: string) => {
    setLoadingNodes((prev) => new Set(prev).add(parentCode))

    try {
      const response = await fetch(
        `/api/kpved/hierarchy?parent=${parentCode}`
      )
      if (!response.ok) throw new Error('Failed to load child nodes')

      const data = await response.json()
      const childNodes = data.nodes || []

      // Update hierarchy with loaded children
      setHierarchy((prev) => {
        const updateNode = (nodes: KPVEDNode[]): KPVEDNode[] => {
          return nodes.map((node) => {
            if (node.code === parentCode) {
              return { ...node, children: childNodes, has_children: childNodes.length > 0 }
            }
            if (node.children) {
              return { ...node, children: updateNode(node.children) }
            }
            return node
          })
        }
        return updateNode(prev)
      })

      return childNodes
    } catch (error) {
      console.error('Error loading child nodes:', error)
      throw error
    } finally {
      setLoadingNodes((prev) => {
        const newSet = new Set(prev)
        newSet.delete(parentCode)
        return newSet
      })
    }
  }, [])

  const toggleNode = useCallback(
    async (code: string) => {
      const isExpanded = expandedNodes.has(code)

      if (isExpanded) {
        // Collapse
        setExpandedNodes((prev) => {
          const newSet = new Set(prev)
          newSet.delete(code)
          return newSet
        })
      } else {
        // Expand
        setExpandedNodes((prev) => new Set(prev).add(code))

        // Find node in hierarchy
        const findNode = (nodes: KPVEDNode[]): KPVEDNode | null => {
          for (const node of nodes) {
            if (node.code === code) return node
            if (node.children) {
              const found = findNode(node.children)
              if (found) return found
            }
          }
          return null
        }

        const node = findNode(hierarchy)
        if (node && !node.children && node.has_children) {
          await loadChildNodes(code)
        }
      }
    },
    [expandedNodes, hierarchy, loadChildNodes]
  )

  const searchKPVED = useCallback(async (query: string) => {
    if (!query.trim()) {
      setSearchResults([])
      setShowSearchResults(false)
      return
    }

    try {
      const response = await fetch(
        `/api/kpved/search?q=${encodeURIComponent(query)}`
      )
      if (!response.ok) throw new Error('Search failed')

      const data = await response.json()
      setSearchResults(data.results || data || [])
      setShowSearchResults(true)
    } catch (error) {
      console.error('Error searching KPVED:', error)
      setSearchResults([])
      setShowSearchResults(false)
    }
  }, [])

  const loadStats = useCallback(async () => {
    try {
      const response = await fetch(`/api/kpved/stats`)
      if (!response.ok) throw new Error('Failed to load stats')

      const data = await response.json()
      setStats({
        total: data.total_codes || data.total || 0,
        levels: data.max_level || data.levels || 0,
      })
    } catch (error) {
      console.error('Error loading stats:', error)
    }
  }, [])

  const selectNode = useCallback((code: string | null) => {
    setSelectedNode(code)
  }, [])

  const copyCode = useCallback((code: string) => {
    navigator.clipboard.writeText(code)
    setCopiedCode(code)
    setTimeout(() => setCopiedCode(null), 2000)
  }, [])

  const clearSearch = useCallback(() => {
    setSearchQuery('')
    setSearchResults([])
    setShowSearchResults(false)
  }, [])

  const getFilteredNodes = useCallback(
    (nodes: KPVEDNode[]): KPVEDNode[] => {
      if (filterLevel === null) return nodes

      const filterRecursive = (nodesList: KPVEDNode[]): KPVEDNode[] => {
        return nodesList
          .map((node) => {
            if (node.level === filterLevel) {
              return node
            }
            if (node.children) {
              const filteredChildren = filterRecursive(node.children)
              if (filteredChildren.length > 0) {
                return { ...node, children: filteredChildren }
              }
            }
            return null
          })
          .filter((node): node is KPVEDNode => node !== null)
      }

      return filterRecursive(nodes)
    },
    [filterLevel]
  )

  return {
    // State
    hierarchy,
    expandedNodes,
    loadingNodes,
    selectedNode,
    searchQuery,
    searchResults,
    showSearchResults,
    stats,
    filterLevel,
    nodePath,
    copiedCode,

    // Setters
    setHierarchy,
    setSearchQuery,
    setFilterLevel,
    setNodePath,

    // Actions
    loadRootHierarchy,
    loadChildNodes,
    toggleNode,
    searchKPVED,
    loadStats,
    selectNode,
    copyCode,
    clearSearch,
    getFilteredNodes,
  }
}
