<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<div id='accessDetailDiv'>
	<div class="circleIcon placeholder-bt" style="top: 10px;right:40px;" placeholder="<%=rb.getString("GuanBi")%>">		
		<span @click='closeDetail' class="el-icon el-icon-circle-close"></span>
	</div>
	<el-ctable id="accessDetailGrid" ref="ctableAccessDetail" :url='detailUrl' :height="height"
				:row-key="'id'" :query-params="params_detail" pagination="true" :rownumber=true>
				
			<!-- 模糊查询 -- 接入规则 -->
			<template slot="toolbar">
				<span style='font-size:14px;font-weight:bold;margin-left:20px;'><%=rb.getString("JieRuXiangQing")%></span>
				<div class='queryGroup'> 
					<el-input v-model='params_detail_form.searchText' @keyup.enter.native="queryDetail" class='pairgrid-query' placeholder='<%=rb.getString("GuiZeMingCheng")%>'></el-input>
					<i @click='queryDetail' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
				</div>
			</template>

			<!-- 主列表 -->
			<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' min-width="200"  prop="serialNumber" ></el-table-column>
			<el-table-column label='<%=rb.getString("ZhuangTai")%>' min-width="100" prop="status" :formatter="statusFmt"></el-table-column>
			<el-table-column label='<%=rb.getString("GuiZeMingCheng")%>' prop="tempName" min-width="200"></el-table-column>
			<el-table-column label='<%=rb.getString("KongZhiFangShi")%>' prop="controlType" min-width="150"></el-table-column>
			<el-table-column label='<%=rb.getString("ShangBaoZhi")%>' min-width="200" prop="reportValue"></el-table-column>
			<el-table-column label='<%=rb.getString("JiaoYanShiJian")%>'  prop="accessTime" min-width="150"></el-table-column>
		</el-ctable>
</div>
<script>
	new Vue({
		el:"#accessDetailDiv",
		data:{
			height:"100%",
			detailUrl:'${ctx}/son/access/queryAccessStatusInfoPageList.action',
			params_detail:{
				tempId : accessVue.rowDataRule.id,
				searchText:'',
				timeZone:timeZone
			},
			params_detail_form:{
				searchText:""
			}
		},
		methods:{
			queryDetail(){
				Object.assign(this.params_detail,this.params_detail_form);
			},
			statusFmt(row,column,value,index){
				if(value == "1"){
					return '<%=rb.getString("JieShou")%>';
				}else{
					return '<%=rb.getString("JuJue")%>';
				}
			},
			closeDetail(){
				accessVue.$refs.slide.hide();
			}
		}
	})
</script>
