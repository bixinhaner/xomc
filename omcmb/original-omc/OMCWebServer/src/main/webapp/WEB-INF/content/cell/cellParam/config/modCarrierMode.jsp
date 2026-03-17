<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
    <li class="paramFor1588">
        <label for="CARRIER_MODE_name">${CARRIER_MODE_name }</label>
        <select id="CARRIER_MODE_name" name="CARRIER_MODE" class="border border-box" onblur="createMML();">
            <option value=""></option>
            <option value="1">Single Carrier</option>
            <option value="2">Dual Carrier</option>
        </select>
    </li>
</ul>

<script type ="text/javascript"></script>