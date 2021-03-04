package css

import (
	"errors"
	"fmt"

	"github.com/npillmayer/tyse/engine/dom/style"
)

// GetCascadedProperty gets the value of a property. The search cascades to
// parent property maps, if available.
//
// Clients will usually call GetProperty(…) instead as this will respect
// CSS semantics for inherited properties.
//
// The call to GetCascadedProperty will flag an error if the style property
// isn't found (which should not happen, as every property should be included
// in the 'user-agent' default style properties).
func GetCascadedProperty(node style.Styler, key string) (style.Property, error) {
	// key has to be found in a property group of type G.
	// For cascading, we will start at the currenty style-tree node and walk
	// upwards until we find a node with a property-group G attached.
	// This upward-traversal must succeed if the property is correctly initialized
	// at least in the user-agent styles.
	// Then, starting with G, we will upward-cascade until key is found.
	groupname := style.GroupNameFromPropertyKey(key)
	var group *style.PropertyGroup
	for node != nil && group == nil {
		group = node.Styles().Group(groupname)
		node = node.StylesCascade()
	}
	if group == nil {
		errmsg := fmt.Sprintf("Cannot find ancestor with prop-group %s -- did you create global properties?", groupname)
		return style.NullStyle, errors.New(errmsg)
	}
	p, _ := group.Cascade(key).Get(key)
	return p, nil // must succeed
}

// GetProperty gets the value of a property. If the property is not set
// locally on the style node and the property is inheritable, he search
// cascades to parent property maps, if available.
//
// The call to GetProperty will flag an error if the style property isn't found
// (which should not happen, as every property should be included in the
// 'user-agent' default style properties).
func GetProperty(node style.Styler, key string) (style.Property, error) {
	if style.IsCascading(key) {
		return GetCascadedProperty(node, key)
	}
	//T().Debugf("css get property: %s is not inherited", key)
	p := GetLocalProperty(node.Styles(), key)
	if p == style.NullStyle {
		p = style.GetUserAgentDefaultProperty(node, key)
	}
	//T().Debugf("css get property: local property value = %+v", p)
	return p, nil
}

// GetLocalProperty returns a style property value, if it is set locally
// for a styled node's property map. No cascading is performed.
func GetLocalProperty(pmap *style.PropertyMap, key string) style.Property {
	groupname := style.GroupNameFromPropertyKey(key)
	var group *style.PropertyGroup
	group = pmap.Group(groupname)
	if group == nil {
		return style.NullStyle
	}
	p, _ := group.Get(key)
	return p
}
