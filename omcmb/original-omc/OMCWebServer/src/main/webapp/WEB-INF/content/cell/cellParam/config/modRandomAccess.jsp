<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_SON_PRACH_CONFIG_INDEX_LIST_name">${LTE_SON_PRACH_CONFIG_INDEX_LIST_name }</label>
		<input id="LTE_SON_PRACH_CONFIG_INDEX_LIST_name" name="LTE_SON_PRACH_CONFIG_INDEX_LIST" title="${LTE_SON_PRACH_CONFIG_INDEX_LIST_title }" class="border border-box" 
			min_value="0" max_value="63" onblur="validateMaxAndMinValSplit(event);createMML();"/>
		<div id="LTE_SON_PRACH_CONFIG_INDEX_LIST_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_SON_PRACH_CONFIG_INDEX_LIST_title }
		</div>
	</li>
	<li>
		<label for="LTE_SON_PRACH_FREQ_OFFSET_LIST_name">${LTE_SON_PRACH_FREQ_OFFSET_LIST_name }</label>
		<input id="LTE_SON_PRACH_FREQ_OFFSET_LIST_name" name="LTE_SON_PRACH_FREQ_OFFSET_LIST" title="${LTE_SON_PRACH_FREQ_OFFSET_LIST_title }" class="border border-box" 
			min_value="1" max_value="92" onblur="validateMaxAndMinValSplit(event);createMML();"/>
		<div id="LTE_SON_PRACH_FREQ_OFFSET_LIST_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_SON_PRACH_FREQ_OFFSET_LIST_title }
		</div>
	</li>
	<li>
		<label for="LTE_SON_PRACH_ZERO_CORRELATIONZONE_CONFIG_LIST_name">${LTE_SON_PRACH_ZERO_CORRELATIONZONE_CONFIG_LIST_name }</label>
		<input id="LTE_SON_PRACH_ZERO_CORRELATIONZONE_CONFIG_LIST_name" name="LTE_SON_PRACH_ZERO_CORRELATIONZONE_CONFIG_LIST" title="${LTE_SON_PRACH_ZERO_CORRELATIONZONE_CONFIG_LIST_title }" class="border border-box" 
			min_value="0" max_value="15" onblur="validateMaxAndMinValSplit(event);createMML();"/>
		<div id="LTE_SON_PRACH_ZERO_CORRELATIONZONE_CONFIG_LIST_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_SON_PRACH_ZERO_CORRELATIONZONE_CONFIG_LIST_title }
		</div>
	</li>
	<%-- <li>
		<label for="LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST_name">${LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST_name}</label>
		<input id="LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST_name" name="LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST" title="${LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST_title }" class="border border-box" 
			min_value="0" max_value="837" onblur="validateMaxAndMinValSplit(event);createMML();"/>
		<div id="LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST_title}
		</div>
	</li> --%>
</ul>