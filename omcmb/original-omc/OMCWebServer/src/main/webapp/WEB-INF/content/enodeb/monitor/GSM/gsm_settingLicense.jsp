<%@ page import="java.util.Locale"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page contentType="text/html;charset=UTF-8"%>

<div id="gsmSetLicensePage" class='commonWarp' style='background: #FFFFFF;width: calc(100% - 2px); height: 100%;'>
	<div class='commonFlex' style='padding: 30px 20px 20px;'>
		<div class='commonItem' style='margin-right: 20px;'>
			<span class='commonTemplateText12' style='display: inline-block; width: 110px;'><%=rb.getString("LicenseBanBen")%></span>
			<span class='commonGeneral12' v-html='version'></span>
		</div>
		<div class='commonItem'>
			<span class='commonTemplateText12' style='display: inline-block; width: 110px;'><%=rb.getString("ShengChengShiJian")%></span>
			<span class='commonGeneral12' v-html='generateDate'></span>
		</div>
	</div>
	<div class='tableBox' style='height: 270px; padding: 0 20px;'>
		<el-ctable id='gnbLicenseTable' ref="gnbLicenseTable" :data='tableData' height='100%' :pagination="false" :rownumber=false :row-key="'id'">
	        <template slot="toolbar">
	         	<div class='commonSize14' style='padding-bottom:8px;'><%=rb.getString("NengLiLieBiao")%></div>             
	        </template>
	        <el-table-column prop="id" label='<%=rb.getString("TeXingID")%>'></el-table-column>
			<el-table-column prop="description" label='<%=rb.getString("MiaoShu")%>'></el-table-column>
			<el-table-column prop="capa_value" label='<%=rb.getString("ShuLiang")%>'></el-table-column>    
			<!--有效期为0 时，显示永久-->
			<el-table-column prop="valid_period" label='<%=rb.getString("YouXiaoQi")%>'>
				<template slot-scope="scope">
					<div v-if="scope.row.valid_period == '0'"><%=rb.getString("Yongjiu")%></div>
					<div v-else>{{scope.row.valid_period}}</div>
				</template>
			</el-table-column>
			<!-- 如果有效期是0，剩余天数则显示永久； 反之 剩余天数则取值  remaining_period-->
			<el-table-column prop="remaining_period" label='<%=rb.getString("ShengYuShiJian")%>'>
				<template slot-scope="scope">
					<div v-if="scope.row.valid_period == '0'"><%=rb.getString("Yongjiu")%></div>
					<div v-else>{{scope.row.remaining_period}}</div>
				</template>
			</el-table-column>                      
		</el-ctable>
	</div>
</div>

<script type="text/javascript">
	var gsmSetLicenseVue = new Vue({
	    el: '#gsmSetLicensePage',
	    data() {
	    	return {
	    		version: '',
	    		generateDate: '',
				tableData: [],
                smallCellCode:'',
                rowDataInfo:'',
	    	}
	    },
	    methods: {
            init(row){
                var vm = this;
                vm.rowDataInfo = row;
                vm.smallCellCode = row.small_cell_code;
                vm.licenseInit();
            },
	    	licenseInit(){
				var vm = this, 
                    small_cell_code = vm.smallCellCode,
					params ={
						small_cell_code: small_cell_code
					};
				// 获取license信息
				setTimeout(function(){
					axios.post('${ctx}/cell/license/getHalobLicenseInfo.action', stringify(params)).then(function(respond){
						var data = respond.data;
						
						if(!vm.isEmptyObject(data)){
							vm.version = data.version;
							vm.generateDate = data.generate_date;
							
							if(data.capacity_list && data.capacity_list.length > 0){
								vm.tableData = data.capacity_list;
							}else{
								vm.tableData = [];
							}	
						}
					}).catch(function(error){});
				},300)
			},
			isEmptyObject(obj){
				for(var key in obj){
					return false;
				}
				return true;
			}
	    },
		mounted(){
	    	eventBus.$off("gsm-data").$on("gsm-data",this.init)
	    }
	});
	
</script> 