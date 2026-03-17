<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<%@ include file="../base.jsp"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_SMALLCELL_START_PCI_name">${LTE_SMALLCELL_START_PCI_name }</label>
		<input id="LTE_SMALLCELL_START_PCI_name" name="LTE_SMALLCELL_START_PCI" title="${LTE_SMALLCELL_START_PCI_title }" class="border border-box" 
			min_value="0" max_value="503" onblur="validateMaxAndMinVal(event);validateSum(event);createMML();" must="1"/>
		<div id="LTE_SMALLCELL_START_PCI_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_SMALLCELL_START_PCI_title }<%=rb.getString("DouHao") %><%=rb.getString("BiTian") %>
		</div>
	</li>
	<li>
		<label for="LTE_SMALLCELL_PCI_RANGE_name">${LTE_SMALLCELL_PCI_RANGE_name }</label>
		<input id="LTE_SMALLCELL_PCI_RANGE_name" name="LTE_SMALLCELL_PCI_RANGE" title="${LTE_SMALLCELL_PCI_RANGE_title }" class="border border-box" 
			min_value="0" max_value="503" onblur="validateMaxAndMinVal(event);validateSum(event);createMML();" must="1"/>
		<div id="LTE_SMALLCELL_PCI_RANGE_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_SMALLCELL_PCI_RANGE_title }<%=rb.getString("DouHao") %><%=rb.getString("BiTian") %>
		</div>
	</li>
</ul>
<script>
	function validateSum(evt) {
		var spci = $('#LTE_SMALLCELL_START_PCI_name'),
			rpci = $('#LTE_SMALLCELL_PCI_RANGE_name'),
			tar = $(evt['target']),
			sum = spci.val()*1 + rpci.val()*1,
			curVal = tar.val().trim(),
			reg = /^[0-9]+$/;
		
		if(reg.test(curVal) && sum<=503) {
			tar.removeClass('err_border');
			$('#'+tar.attr('id')+'_err').removeClass('redColor');
		}else if(curVal){
			tar.addClass('err_border');
			$('#'+tar.attr('id')+'_err').addClass('redColor');
		}
	}
</script>