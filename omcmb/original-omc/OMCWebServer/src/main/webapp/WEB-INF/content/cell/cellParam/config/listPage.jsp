<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<%@ taglib uri="http://java.sun.com/jsp/jstl/core" prefix="c" %>
<%@ include file="../base.jsp"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li id="cell_index_li" style="display: none">
		<label for="i_name">RU Index</label>
		<input id="i_name" name="i" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="Integer,range:1-32" 
			min_value="1" max_value="32" class="border border-box" must="0"/>
		<div id="i_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			Integer,range:1-32
		</div>
	</li>
	<c:if test='${hardwareVersion == "Nova430i_CA" || hardwareVersion == "Nova430_CA" || hardwareVersion == "Nova430i_DC" || hardwareVersion == "Nova430_DC" || hardwareVersion == "Nova430i_SC" }'>
		<div id="checkboxForIndex">
			<input value="1" type="checkbox" style="width: 20px;float: left;margin: 6px 0 5px 0;"/>
			<label style="margin-bottom: 5px;"><%=rb.getString("XiaoQuBianHao")%>1</label>
			<input value="2" type="checkbox" style="width: 20px;float: left;margin: 6px 0 5px 0;"/>
			<label style="margin-bottom: 5px;"><%=rb.getString("XiaoQuBianHao")%>2</label>
		</div>
	</c:if>
</ul>

<script type="text/javascript">
	if ("${isShowIndex}" == "true"){
		$("#cell_index_li").css("display","inline-block")
	}
</script>