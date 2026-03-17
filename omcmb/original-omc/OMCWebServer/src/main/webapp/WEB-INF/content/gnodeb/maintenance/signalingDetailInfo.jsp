<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<script type="text/javascript">
	var ctx = "${ctx}";
</script>
<style>
	#showGnbDate span[name='R']{
	    display:inline-block;
		width:30px;
	}
	#showGnbDate span[name='N']{
	    display:inline-block;
		margin-right:20px;
		width:30px;
		color:rgb(156,156,156);
		background:rgb(240,240,240);
	}
	#showGnbDate span[name='R'] span:hover{
	    background:#88bfd6;
	    cursor:default;
	}
	#showGnbDate .chosen{
		background:#c4e6f5;
	}
</style>
<div class="easyui-layout" data-options="border:false,fit:true">
	<div region="center" id="gnbCtrlScroll" data-options="border:false" >
	<ul id="testGnbTree" style="padding:20px;position:relative;"></ul>
	</div>
	<div region="south" data-options="height:300,minHeight:150,maxHeight:400,split:true," style="">
		<div id="showGnbDate" style="padding:20px;">		
	</div>
	</div>
</div>

<script type="text/javascript">
$(function() {
	$("#testGnbTree").tree({
		animate: false,
		lines: true,
		border: true,
		onlyLeafCheck: true,
        onDblClick: function(node) {
        	if (!$(this).tree("isLeaf", node["target"])) {
        		$(this).tree("toggle", node["target"]);
        	}
        },
        onClick:function(node){
        	$("#showGnbDate span[name='R']").removeClass("chosen").css({'width':'30px','margin-right':'0px'});
        	$("#showGnbDate span[name='R']").each(function (index,element){
        		
        		if( parseInt($(element).attr("pos"))>=parseInt(node.pos) &&  
        				parseInt($(element).attr("pos"))<(parseInt(node.pos)+parseInt(node.size))){
        			if(parseInt($(element).attr("pos"))==(parseInt(node.pos)+parseInt(node.size))-1){
        				$(element).css({'width':'15px','margin-right':'15px'});
        			}
        			if((parseInt($(element).attr("pos"))-(${begin_position}-1))%16==0){
        				$(element).css({'width':'15px','margin-right':'15px'});
        			}
        			$(element).addClass("chosen");
        		}
        	});
        },
        onLoadSuccess:function(node,data){
        	//$("#testGnbTree").tree("collapseAll");
            $("#showGnbDate").append( ${hexData} );
            $("#showGnbDate span[name='R']").each(function(){
            	$(this).click(function(){
            		/* $("#testGnbTree").tree("collapseAll"); */
            		var search_pos = $(this).attr("pos");
            		var roots = data;
            		searchTree(roots,search_pos) 
					
             		function searchTree(parentNode,searchPos){
            			for(var i=0;i<parentNode.length;i++){

            				if($("#testGnbTree").tree('isLeaf',parentNode[i].target) ){
            					if(searchPos > parentNode[i].pos-1 && searchPos < parseInt(parentNode[i].pos) + parseInt(parentNode[i].size) ){
            						parentNode[i].target.click();
            						$("#testGnbTree").tree('expandTo',parentNode[i].target);
            						$("#testGnbTree").tree('select',parentNode[i].target);
            						scrol()            						
            					}
            					
            					
            				}else if(parentNode[i].pos == ${begin_position} || parentNode[i].pos > ${begin_position} ){
            					
            						if(parentNode[i].pos-1 < searchPos && $("#testGnbTree").tree('getChildren',parentNode[i].target)[0].pos > searchPos){
            							parentNode[i].target.click();
            							$("#testGnbTree").tree('expandTo',parentNode[i].target);
            							$("#testGnbTree").tree('select',parentNode[i].target);
            							scrol()
            							
            						}
            						
            						searchTree(parentNode[i].children,search_pos)
            				}
            			}
            		} 
            		
            		function scrol(){
            			var toTop = $("#testGnbTree div").hasClass("tree-node-selected");
                		var isT = $("#testGnbTree div.tree-node-selected").position().top;
                		if(toTop){
                			$("#gnbCtrlScroll").scrollTop(isT-200);
                		}
            		}
            		
            		
            	})
            })
        }
	}).tree("loadData", ${treeData}); 
});

</script>