<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="IPSEC_ENABLE_name">${IPSEC_ENABLE_name }</label>
		<select id="IPSEC_ENABLE_name" name="IPSEC_ENABLE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">true</option>
			<option value="0">false</option>
		</select>
	</li>
	<li>
		<label for="IPSEC_RIGHTIKEPORT_name">${IPSEC_RIGHTIKEPORT_name }</label>
		<select id="IPSEC_RIGHTIKEPORT_name" name="IPSEC_RIGHTIKEPORT" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="4500">4500</option>
		</select>
	</li>
	<li>
		<label for="LEFT_INTERFACE_name">${LEFT_INTERFACE_name }</label>
		<select id="LEFT_INTERFACE_name" name="LEFT_INTERFACE" class="border border-box none-valid" onblur="createMML();">
			<option value="eth2">WAN(eth2)</option>
			<option value="pppoe-wan">PPPOE(pppoe-wan)</option>
			<option value="">none</option>
		</select>
	</li>
</ul>
