<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<%@ include file="../base.jsp"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="DnsRuleIndex_name"><%=rb.getString("CeLueSuoYin")%></label>
		<input id="DnsRuleIndex_name" name="DnsRuleIndex" type="text" must="1" min_value="1" max_value="128"
			onblur="validateMaxAndMinVal(event);" class="border border-box root-param" title="Integer,range: 1-128"/>
		<div id="DnsRuleIndex_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			<%=rb.getString("DnsRuleIndexNameErr") %>
		</div>
	</li>
</ul>